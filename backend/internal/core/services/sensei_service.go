package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"godojo/internal/adapters/chatstore"
	"godojo/internal/core"
	"godojo/internal/core/domain"
	"godojo/internal/core/ports"
)

const (
	maxAgentRounds       = 5
	agentTimeout         = 60 * time.Second
	statusChannelBufSize = 10
)

var senseiToolIntentHints = []string{
	"archivo",
	"workspace",
	"roadmap",
	"seccion",
	"sección",
	"tema",
	"topic",
	"ejercicio",
	"ejercicios",
	"crea",
	"creá",
	"crear",
	"armame",
	"armá",
	"revisa",
	"revisá",
	"revisar",
	"lee",
	"leé",
	"leer",
	"lista",
	"listá",
	"listar",
	"ultimo archivo",
	"último archivo",
	"termine",
	"terminé",
	"complete",
	"completé",
}

type senseiRunMetrics struct {
	startedAt     time.Time
	rounds        int
	providerCalls int
	toolCalls     int
}

func newSenseiRunMetrics() senseiRunMetrics {
	return senseiRunMetrics{startedAt: time.Now()}
}

func (m senseiRunMetrics) statusLine() string {
	elapsed := time.Since(m.startedAt).Round(10 * time.Millisecond)
	return fmt.Sprintf(
		"Métricas: %s · %d %s · %d %s al modelo · %d %s",
		elapsed,
		m.rounds,
		pluralizeSpanish(m.rounds, "ronda", "rondas"),
		m.providerCalls,
		pluralizeSpanish(m.providerCalls, "llamada", "llamadas"),
		m.toolCalls,
		pluralizeSpanish(m.toolCalls, "herramienta", "herramientas"),
	)
}

// SenseiService orchestrates the agentic AI loop.
// It sends messages to the AI, executes function calls locally,
// and feeds results back until a final text response is produced.
type SenseiService struct {
	provider            ports.SenseiProvider
	tools               *core.ToolRegistry
	workspace           ports.WorkspaceManager
	roadmapSvc          *RoadmapService
	mu                  sync.RWMutex
	currentTopicSlug    string
	recentWorkspaceFile string
}

// NewSenseiService creates a SenseiService and registers the default tools.
func NewSenseiService(
	provider ports.SenseiProvider,
	tools *core.ToolRegistry,
	workspace ports.WorkspaceManager,
	roadmap *RoadmapService,
) *SenseiService {
	svc := &SenseiService{
		provider:   provider,
		tools:      tools,
		workspace:  workspace,
		roadmapSvc: roadmap,
	}

	// Register built-in tools
	svc.registerCreateExerciseFile()
	svc.registerReadRoadmapSection()
	svc.registerListWorkspaceFiles()
	svc.registerReadWorkspaceFile()
	svc.registerReadRecentWorkspaceFile()

	return svc
}

// ProcessMessage handles a user message through the agent loop.
// Returns the final text response and a buffered status channel with progress updates.
// The status channel is pre-populated and closed when ProcessMessage returns.
func (s *SenseiService) ProcessMessage(ctx context.Context, systemPrompt string, userMessage string, session *chatstore.ChatSession) (response string, statusUpdates <-chan string, err error) {
	// Append user message to session
	now := time.Now()
	session.Messages = append(session.Messages, chatstore.ChatMessage{
		Role:    "user",
		Content: userMessage,
		Time:    now,
	})

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, agentTimeout)
	defer cancel()

	// Internal status channel
	internalCh := make(chan string, statusChannelBufSize)

	go func() {
		defer close(internalCh)
		s.runAgentLoop(ctx, systemPrompt, userMessage, session, internalCh)
	}()

	// Collect statuses and final response
	var statuses []string
	for s := range internalCh {
		if strings.HasPrefix(s, "done:") {
			response = strings.TrimPrefix(s, "done:")
			statuses = append(statuses, "done")
		} else if strings.HasPrefix(s, "error:") {
			response = strings.TrimPrefix(s, "error:")
			statuses = append(statuses, "Error: "+response)
		} else {
			statuses = append(statuses, s)
		}
	}

	// Return pre-populated closed channel
	resultCh := make(chan string, len(statuses))
	for _, s := range statuses {
		resultCh <- s
	}
	close(resultCh)

	return response, resultCh, nil
}

// runAgentLoop executes the agent loop: send → parse → execute tools → repeat.
func (s *SenseiService) runAgentLoop(ctx context.Context, systemPrompt string, userMessage string, session *chatstore.ChatSession, statusCh chan<- string) {
	toolDeclarations := s.initialToolDeclarations(userMessage)
	metrics := newSenseiRunMetrics()

	for round := 0; round < maxAgentRounds; round++ {
		select {
		case <-ctx.Done():
			statusCh <- metrics.statusLine()
			statusCh <- "error:Se agotó el tiempo. Reformulá la pregunta."
			return
		default:
		}

		// Send "Pensando..." status
		statusCh <- "Pensando..."
		metrics.rounds++
		metrics.providerCalls++

		parts, err := s.provider.SendMessage(ctx, systemPrompt, session.Messages, toolDeclarations)
		if err != nil {
			statusCh <- metrics.statusLine()
			statusCh <- fmt.Sprintf("error:Error del sensei: %v", err)
			return
		}

		for {
			// Separate text and functionCall parts
			var textParts []string
			var functionCalls []domain.FunctionCall

			for _, part := range parts {
				if part.FunctionCall != nil {
					functionCalls = append(functionCalls, *part.FunctionCall)
				}
				if part.Text != "" {
					textParts = append(textParts, part.Text)
				}
			}

			// If no function calls → this is the final response
			if len(functionCalls) == 0 {
				finalText := strings.Join(textParts, "\n")
				if finalText == "" {
					finalText = "El sensei no tiene respuesta para eso. ¿Querés reformular la pregunta?"
				}

				// Append sensei response to session
				session.Messages = append(session.Messages, chatstore.ChatMessage{
					Role:    "sensei",
					Content: finalText,
					Time:    time.Now(),
				})

				statusCh <- metrics.statusLine()
				statusCh <- "done:" + finalText
				return
			}

			fc := functionCalls[0]
			statusCh <- fmt.Sprintf("Ejecutando %s...", fc.Name)
			metrics.toolCalls++

			// Execute the tool
			result, toolErr := s.tools.Execute(fc.Name, fc.Args)

			var responsePayload interface{}
			if toolErr != nil {
				responsePayload = map[string]interface{}{
					"error": toolErr.Error(),
				}
			} else {
				responsePayload = result
			}

			// Append the model's functionCall message (as sensei/model role with functionCall content)
			session.Messages = append(session.Messages, chatstore.ChatMessage{
				Role:    "sensei",
				Content: fmt.Sprintf("[functionCall: %s]", fc.Name),
				Time:    time.Now(),
			})

			// Append the functionResponse as a user message for local session persistence.
			session.Messages = append(session.Messages, chatstore.ChatMessage{
				Role:    "user",
				Content: fmt.Sprintf("[functionResponse: %s -> %v]", fc.Name, responsePayload),
				Time:    time.Now(),
			})

			if metrics.rounds >= maxAgentRounds {
				statusCh <- metrics.statusLine()
				statusCh <- "done:Lo siento, tardé mucho. reformulá la pregunta."
				return
			}

			statusCh <- "Pensando..."
			metrics.rounds++
			metrics.providerCalls++
			parts, err = s.provider.SendFunctionResponse(ctx, session.Messages, fc.ID, fc.Name, responsePayload)
			if err != nil {
				statusCh <- metrics.statusLine()
				statusCh <- fmt.Sprintf("error:Error del sensei: %v", err)
				return
			}
		}
	}

	// Max rounds exhausted
	statusCh <- metrics.statusLine()
	statusCh <- "done:Lo siento, tardé mucho. reformulá la pregunta."
}

func (s *SenseiService) initialToolDeclarations(userMessage string) []domain.ToolDeclaration {
	if !shouldStartWithTools(userMessage) {
		return nil
	}
	return s.tools.GetDeclarations()
}

func shouldStartWithTools(userMessage string) bool {
	normalized := strings.ToLower(strings.TrimSpace(userMessage))
	if normalized == "" {
		return false
	}

	for _, hint := range senseiToolIntentHints {
		if strings.Contains(normalized, hint) {
			return true
		}
	}

	return false
}

func pluralizeSpanish(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// registerCreateExerciseFile registers the create_exercise_file tool.
func (s *SenseiService) registerCreateExerciseFile() {
	s.tools.Register("create_exercise_file", domain.ToolDeclaration{
		Name:        "create_exercise_file",
		Description: "Crea un archivo .go o una ruta relativa dentro del workspace del estudiante. Usala cuando el estudiante pida practicar un concepto o crear un ejercicio. Si hay un topic_slug, guardá los archivos dentro de esa carpeta temática. El archivo debería ser un scaffold con pistas, no una solución completa, salvo que te la pidan explícitamente.",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"filename":   {Type: "STRING", Description: "Ruta relativa del archivo; puede incluir subcarpetas y debe terminar en .go"},
				"topic_slug": {Type: "STRING", Description: "Slug del tema para agrupar archivos en una carpeta temática"},
				"content":    {Type: "STRING", Description: "Contenido del archivo Go"},
			},
			Required: []string{"filename", "content"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		filename, _ := args["filename"].(string)
		topicSlug, _ := args["topic_slug"].(string)
		content, _ := args["content"].(string)

		if topicSlug == "" {
			topicSlug = s.getCurrentTopicSlug()
		} else {
			if _, err := s.roadmapSvc.GetTopicBySlug(topicSlug); err != nil {
				return nil, err
			}
		}

		relativePath := filename
		if topicSlug != "" {
			relativePath = joinTopicPath(topicSlug, filename)
			s.setCurrentTopicSlug(topicSlug)
		}

		if err := s.workspace.CreateFile(relativePath, content); err != nil {
			return nil, err
		}
		s.setRecentWorkspaceFile(relativePath)

		return map[string]interface{}{
			"filename":   relativePath,
			"topic_slug": topicSlug,
			"success":    true,
			"message":    fmt.Sprintf("Archivo %q creado en tu workspace.", relativePath),
		}, nil
	})
}

// registerReadRoadmapSection registers the read_roadmap_section tool.
func (s *SenseiService) registerReadRoadmapSection() {
	s.tools.Register("read_roadmap_section", domain.ToolDeclaration{
		Name:        "read_roadmap_section",
		Description: "Lee una sección del roadmap de GoDojo. Usala para contextualizar tus respuestas con el plan de estudios.",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"section_slug": {Type: "STRING", Description: "Slug de la sección (ej: 'fase-1', 'variables', 'goroutines')"},
			},
			Required: []string{"section_slug"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		slug, _ := args["section_slug"].(string)

		// Try as phase first, then as topic
		phase, phaseErr := s.roadmapSvc.GetPhaseBySlug(slug)
		if phaseErr == nil {
			topicNames := make([]string, len(phase.Topics))
			for i, topic := range phase.Topics {
				topicNames[i] = topic.Title
			}
			return map[string]interface{}{
				"type":   "phase",
				"id":     phase.ID,
				"title":  phase.Title,
				"topics": topicNames,
			}, nil
		}

		topic, topicErr := s.roadmapSvc.GetTopicBySlug(slug)
		if topicErr == nil {
			s.setCurrentTopicSlug(topic.Slug)
			return map[string]interface{}{
				"type":             "topic",
				"slug":             topic.Slug,
				"title":            topic.Title,
				"summary":          topic.Description,
				"workspace_folder": topic.Slug,
			}, nil
		}

		return nil, fmt.Errorf("sección %q no encontrada (ni fase ni tema)", slug)
	})
}

func (s *SenseiService) registerListWorkspaceFiles() {
	s.tools.Register("list_workspace_files", domain.ToolDeclaration{
		Name:        "list_workspace_files",
		Description: "Lista los archivos .go que existen en el workspace del estudiante, incluyendo subcarpetas temáticas. Usala cuando el usuario pregunte qué ejercicios, archivos o lecciones tiene disponibles.",
		Parameters: domain.ToolParameters{
			Type:       "OBJECT",
			Properties: map[string]domain.ToolProperty{},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		files, err := s.workspace.ListFiles()
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"count":          len(files),
			"files":          files,
			"workspace_path": s.workspace.WorkspacePath(),
			"recent_file":    s.getRecentWorkspaceFile(),
		}, nil
	})
}

func (s *SenseiService) registerReadWorkspaceFile() {
	s.tools.Register("read_workspace_file", domain.ToolDeclaration{
		Name:        "read_workspace_file",
		Description: "Lee el contenido de un archivo .go del workspace del estudiante. Usala después de listar archivos o cuando el usuario pida revisar un ejercicio o archivo específico.",
		Parameters: domain.ToolParameters{
			Type: "OBJECT",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "STRING", Description: "Ruta relativa del archivo .go dentro del workspace"},
			},
			Required: []string{"filename"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		filename, _ := args["filename"].(string)
		content, err := s.workspace.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		s.setRecentWorkspaceFile(filename)

		return map[string]interface{}{
			"filename": filename,
			"content":  content,
		}, nil
	})
}

func (s *SenseiService) registerReadRecentWorkspaceFile() {
	s.tools.Register("read_recent_workspace_file", domain.ToolDeclaration{
		Name:        "read_recent_workspace_file",
		Description: "Lee el archivo .go más reciente en el que estuvieron trabajando en esta sesión. Usala cuando el usuario diga 'ya terminé', 'revisalo', o pida feedback del último ejercicio sin repetir el nombre del archivo.",
		Parameters: domain.ToolParameters{
			Type:       "OBJECT",
			Properties: map[string]domain.ToolProperty{},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		filename := s.getRecentWorkspaceFile()
		if filename == "" {
			files, err := s.workspace.ListFiles()
			if err != nil {
				return nil, err
			}
			if len(files) == 1 {
				filename = files[0]
			}
		}
		if filename == "" {
			return nil, fmt.Errorf("no hay un archivo reciente identificado para revisar")
		}

		content, err := s.workspace.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		s.setRecentWorkspaceFile(filename)

		return map[string]interface{}{
			"filename": filename,
			"content":  content,
		}, nil
	})
}

func (s *SenseiService) setCurrentTopicSlug(slug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentTopicSlug = slug
}

func (s *SenseiService) getCurrentTopicSlug() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentTopicSlug
}

func (s *SenseiService) setRecentWorkspaceFile(filename string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentWorkspaceFile = filename
}

func (s *SenseiService) getRecentWorkspaceFile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recentWorkspaceFile
}

func joinTopicPath(topicSlug string, filename string) string {
	normalizedTopic := strings.Trim(strings.ReplaceAll(topicSlug, "\\", "/"), "/")
	normalizedFilename := strings.TrimLeft(strings.ReplaceAll(filename, "\\", "/"), "/")
	if normalizedTopic == "" || normalizedFilename == "" {
		return normalizedFilename
	}
	if normalizedFilename == normalizedTopic || strings.HasPrefix(normalizedFilename, normalizedTopic+"/") {
		return normalizedFilename
	}
	return normalizedTopic + "/" + normalizedFilename
}
