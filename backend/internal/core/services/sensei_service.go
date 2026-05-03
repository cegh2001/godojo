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

// SenseiService orchestrates the agentic AI loop.
// It sends messages to the AI, executes function calls locally,
// and feeds results back until a final text response is produced.
type SenseiService struct {
	provider         ports.SenseiProvider
	tools            *core.ToolRegistry
	workspace        ports.WorkspaceManager
	roadmapSvc       *RoadmapService
	mu               sync.RWMutex
	currentTopicSlug string
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
		s.runAgentLoop(ctx, systemPrompt, session, internalCh)
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
func (s *SenseiService) runAgentLoop(ctx context.Context, systemPrompt string, session *chatstore.ChatSession, statusCh chan<- string) {
	toolDeclarations := s.tools.GetDeclarations()

	for round := 0; round < maxAgentRounds; round++ {
		select {
		case <-ctx.Done():
			statusCh <- "error:Se agotó el tiempo. Reformulá la pregunta."
			return
		default:
		}

		// Send "Pensando..." status
		statusCh <- "Pensando..."

		parts, err := s.provider.SendMessage(ctx, systemPrompt, session.Messages, toolDeclarations)
		if err != nil {
			statusCh <- fmt.Sprintf("error:Error del sensei: %v", err)
			return
		}

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

			statusCh <- "done:" + finalText
			return
		}

		// Execute each function call and build functionResponse messages
		for _, fc := range functionCalls {
			statusCh <- fmt.Sprintf("Ejecutando %s...", fc.Name)

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

			// Append the functionResponse as a user message
			session.Messages = append(session.Messages, chatstore.ChatMessage{
				Role:    "user",
				Content: fmt.Sprintf("[functionResponse: %s -> %v]", fc.Name, responsePayload),
				Time:    time.Now(),
			})
		}
	}

	// Max rounds exhausted
	statusCh <- "done:Lo siento, tardé mucho. reformulá la pregunta."
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
