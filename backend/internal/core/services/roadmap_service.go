package services

import (
	"fmt"

	"godojo/internal/core/domain"
)

// RoadmapService provides access to the hardcoded learning roadmap structure.
type RoadmapService struct {
	roadmap *domain.Roadmap
	// flat list of all topics across phases for easy lookup and ordering
	allTopics []*domain.Topic
}

// NewRoadmapService creates a RoadmapService with the hardcoded MVP roadmap.
func NewRoadmapService() *RoadmapService {
	fase1 := buildFase1()
	fase2 := buildFase2()

	rm, err := domain.NewRoadmap([]*domain.Phase{fase1, fase2})
	if err != nil {
		// This is a programming error — the hardcoded roadmap should always be valid.
		panic(fmt.Sprintf("hardcoded roadmap is invalid: %v", err))
	}

	// Build flat topic list
	var allTopics []*domain.Topic
	allTopics = append(allTopics, fase1.Topics...)
	allTopics = append(allTopics, fase2.Topics...)

	return &RoadmapService{
		roadmap:   rm,
		allTopics: allTopics,
	}
}

func buildFase1() *domain.Phase {
	topics := []*domain.Topic{
		mustNewTopic("variables", "Variables", "Declaración, inicialización y tipos básicos de variables en Go."),
		mustNewTopic("tipos", "Tipos de Datos", "Tipos numéricos, strings, booleanos y conversiones entre tipos."),
		mustNewTopic("funciones", "Funciones", "Declaración, parámetros, retornos, funciones variádicas y closures."),
		mustNewTopic("packages", "Paquetes", "Organización de código, imports, visibilidad (exported vs unexported)."),
		mustNewTopic("control-de-flujo", "Control de Flujo", "if, for, switch, defer, panic y recover."),
	}
	phase, err := domain.NewPhase("fase-1", "Fundamentos", topics)
	if err != nil {
		panic(fmt.Sprintf("hardcoded fase-1 is invalid: %v", err))
	}
	return phase
}

func buildFase2() *domain.Phase {
	topics := []*domain.Topic{
		mustNewTopic("arrays", "Arrays", "Arrays de tamaño fijo, índices y recorrido con range."),
		mustNewTopic("slices", "Slices", "Slices dinámicos, append, copy, slicing y capacidad vs longitud."),
		mustNewTopic("maps", "Maps", "Mapas clave-valor, creación, inserción, eliminación y verificación de existencia."),
		mustNewTopic("structs", "Structs", "Definición de structs, métodos, embedding y constructores."),
	}
	phase, err := domain.NewPhase("fase-2", "Estructuras de Datos", topics)
	if err != nil {
		panic(fmt.Sprintf("hardcoded fase-2 is invalid: %v", err))
	}
	return phase
}

func mustNewTopic(slug, title, description string) *domain.Topic {
	topic, err := domain.NewTopic(slug, title, description, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("hardcoded topic %q is invalid: %v", slug, err))
	}
	return topic
}

// GetRoadmap returns the full roadmap with both phases.
func (s *RoadmapService) GetRoadmap() (*domain.Roadmap, error) {
	return s.roadmap, nil
}

// GetTopicBySlug finds a topic by its slug across all phases.
func (s *RoadmapService) GetTopicBySlug(slug string) (*domain.Topic, error) {
	if slug == "" {
		return nil, fmt.Errorf("topic slug is required")
	}
	for _, topic := range s.allTopics {
		if topic.Slug == slug {
			return topic, nil
		}
	}
	return nil, fmt.Errorf("topic %q not found", slug)
}

// GetNextTopic returns the next topic in the roadmap order.
// Returns an error if the current slug is the last topic or not found.
func (s *RoadmapService) GetNextTopic(currentSlug string) (*domain.Topic, error) {
	if currentSlug == "" {
		return nil, fmt.Errorf("current slug is required")
	}
	for i, topic := range s.allTopics {
		if topic.Slug == currentSlug {
			if i+1 >= len(s.allTopics) {
				return nil, fmt.Errorf("no next topic after %q (it is the last one)", currentSlug)
			}
			return s.allTopics[i+1], nil
		}
	}
	return nil, fmt.Errorf("topic %q not found", currentSlug)
}

// GetPhaseBySlug finds a phase by its id.
func (s *RoadmapService) GetPhaseBySlug(slug string) (*domain.Phase, error) {
	if slug == "" {
		return nil, fmt.Errorf("phase slug is required")
	}
	for _, phase := range s.roadmap.Phases {
		if phase.ID == slug {
			return phase, nil
		}
	}
	return nil, fmt.Errorf("phase %q not found", slug)
}
