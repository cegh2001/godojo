package services_test

import (
	"testing"

	"godojo/internal/core/services"
)

func TestRoadmapService_GetRoadmap(t *testing.T) {
	svc := services.NewRoadmapService()

	rm, err := svc.GetRoadmap()
	if err != nil {
		t.Fatalf("GetRoadmap() returned error: %v", err)
	}

	if rm == nil {
		t.Fatal("roadmap should not be nil")
	}

	if len(rm.Phases) != 7 {
		t.Errorf("expected 7 phases, got %d", len(rm.Phases))
	}

	// Phase 1: Fundamentos
	fase1 := rm.Phases[0]
	if fase1.ID != "fase-1" {
		t.Errorf("phase 1 id = %q, want %q", fase1.ID, "fase-1")
	}
	if fase1.Title != "Fundamentos" {
		t.Errorf("phase 1 title = %q, want %q", fase1.Title, "Fundamentos")
	}
	if len(fase1.Topics) != 5 {
		t.Errorf("phase 1 should have 5 topics, got %d", len(fase1.Topics))
	}

	// Phase 2: Estructuras de Datos
	fase2 := rm.Phases[1]
	if fase2.ID != "fase-2" {
		t.Errorf("phase 2 id = %q, want %q", fase2.ID, "fase-2")
	}
	if fase2.Title != "Estructuras de Datos" {
		t.Errorf("phase 2 title = %q, want %q", fase2.Title, "Estructuras de Datos")
	}
	if len(fase2.Topics) != 4 {
		t.Errorf("phase 2 should have 4 topics, got %d", len(fase2.Topics))
	}

	// Phase 3: Punteros y Memoria
	fase3 := rm.Phases[2]
	if fase3.ID != "fase-3" {
		t.Errorf("phase 3 id = %q, want %q", fase3.ID, "fase-3")
	}
	if len(fase3.Topics) != 1 {
		t.Errorf("phase 3 should have 1 topic, got %d", len(fase3.Topics))
	}

	// Phase 4: Métodos e Interfaces
	fase4 := rm.Phases[3]
	if fase4.ID != "fase-4" {
		t.Errorf("phase 4 id = %q, want %q", fase4.ID, "fase-4")
	}
	if len(fase4.Topics) != 2 {
		t.Errorf("phase 4 should have 2 topics, got %d", len(fase4.Topics))
	}

	// Phase 5: Manejo de Errores
	fase5 := rm.Phases[4]
	if fase5.ID != "fase-5" {
		t.Errorf("phase 5 id = %q, want %q", fase5.ID, "fase-5")
	}
	if len(fase5.Topics) != 1 {
		t.Errorf("phase 5 should have 1 topic, got %d", len(fase5.Topics))
	}

	// Phase 6: Concurrencia
	fase6 := rm.Phases[5]
	if fase6.ID != "fase-6" {
		t.Errorf("phase 6 id = %q, want %q", fase6.ID, "fase-6")
	}
	if len(fase6.Topics) != 2 {
		t.Errorf("phase 6 should have 2 topics, got %d", len(fase6.Topics))
	}

	// Phase 7: Standard Library
	fase7 := rm.Phases[6]
	if fase7.ID != "fase-7" {
		t.Errorf("phase 7 id = %q, want %q", fase7.ID, "fase-7")
	}
	if len(fase7.Topics) != 1 {
		t.Errorf("phase 7 should have 1 topic, got %d", len(fase7.Topics))
	}
}

func TestRoadmapService_GetTopicBySlug(t *testing.T) {
	svc := services.NewRoadmapService()

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{name: "variables topic", slug: "variables", wantErr: false},
		{name: "tipos topic", slug: "tipos", wantErr: false},
		{name: "funciones topic", slug: "funciones", wantErr: false},
		{name: "packages topic", slug: "packages", wantErr: false},
		{name: "control-de-flujo topic", slug: "control-de-flujo", wantErr: false},
		{name: "arrays topic", slug: "arrays", wantErr: false},
		{name: "slices topic", slug: "slices", wantErr: false},
		{name: "maps topic", slug: "maps", wantErr: false},
		{name: "structs topic", slug: "structs", wantErr: false},
		{name: "non-existent topic", slug: "inexistente", wantErr: true},
		{name: "empty slug", slug: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, err := svc.GetTopicBySlug(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for slug %q but got topic=%+v", tt.slug, topic)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for slug %q: %v", tt.slug, err)
				return
			}
			if topic.Slug != tt.slug {
				t.Errorf("topic slug = %q, want %q", topic.Slug, tt.slug)
			}
		})
	}
}

func TestRoadmapService_GetNextTopic(t *testing.T) {
	svc := services.NewRoadmapService()

	tests := []struct {
		name        string
		currentSlug string
		wantSlug    string
		wantErr     bool
	}{
		{
			name:        "from variables to tipos",
			currentSlug: "variables",
			wantSlug:    "tipos",
			wantErr:     false,
		},
		{
			name:        "from tipos to funciones",
			currentSlug: "tipos",
			wantSlug:    "funciones",
			wantErr:     false,
		},
		{
			name:        "from funciones to packages",
			currentSlug: "funciones",
			wantSlug:    "packages",
			wantErr:     false,
		},
		{
			name:        "from packages to control-de-flujo",
			currentSlug: "packages",
			wantSlug:    "control-de-flujo",
			wantErr:     false,
		},
		{
			name:        "from control-de-flujo to arrays (cross-phase)",
			currentSlug: "control-de-flujo",
			wantSlug:    "arrays",
			wantErr:     false,
		},
		{
			name:        "from arrays to slices",
			currentSlug: "arrays",
			wantSlug:    "slices",
			wantErr:     false,
		},
		{
			name:        "from slices to maps",
			currentSlug: "slices",
			wantSlug:    "maps",
			wantErr:     false,
		},
		{
			name:        "from maps to structs",
			currentSlug: "maps",
			wantSlug:    "structs",
			wantErr:     false,
		},
		{
			name:        "from structs to punteros (cross-phase)",
			currentSlug: "structs",
			wantSlug:    "punteros",
			wantErr:     false,
		},
		{
			name:        "from punteros to metodos",
			currentSlug: "punteros",
			wantSlug:    "metodos",
			wantErr:     false,
		},
		{
			name:        "from metodos to interfaces",
			currentSlug: "metodos",
			wantSlug:    "interfaces",
			wantErr:     false,
		},
		{
			name:        "from interfaces to errores",
			currentSlug: "interfaces",
			wantSlug:    "errores",
			wantErr:     false,
		},
		{
			name:        "from errores to goroutines",
			currentSlug: "errores",
			wantSlug:    "goroutines",
			wantErr:     false,
		},
		{
			name:        "from goroutines to channels",
			currentSlug: "goroutines",
			wantSlug:    "channels",
			wantErr:     false,
		},
		{
			name:        "from channels to stdlib",
			currentSlug: "channels",
			wantSlug:    "stdlib",
			wantErr:     false,
		},
		{
			name:        "last topic has no next",
			currentSlug: "stdlib",
			wantSlug:    "",
			wantErr:     true,
		},
		{
			name:        "non-existent slug",
			currentSlug: "no-existe",
			wantSlug:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := svc.GetNextTopic(tt.currentSlug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q but got topic=%+v", tt.currentSlug, next)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tt.currentSlug, err)
				return
			}
			if next.Slug != tt.wantSlug {
				t.Errorf("next topic = %q, want %q", next.Slug, tt.wantSlug)
			}
		})
	}
}

func TestRoadmapService_GetPhaseBySlug(t *testing.T) {
	svc := services.NewRoadmapService()

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{name: "fase 1", slug: "fase-1", wantErr: false},
		{name: "fase 2", slug: "fase-2", wantErr: false},
		{name: "non-existent", slug: "fase-99", wantErr: true},
		{name: "empty slug", slug: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, err := svc.GetPhaseBySlug(tt.slug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for slug %q but got phase=%+v", tt.slug, phase)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if phase.ID != tt.slug {
				t.Errorf("phase id = %q, want %q", phase.ID, tt.slug)
			}
		})
	}
}

func TestRoadmapService_AllSlugsUnique(t *testing.T) {
	svc := services.NewRoadmapService()
	rm, err := svc.GetRoadmap()
	if err != nil {
		t.Fatalf("GetRoadmap() failed: %v", err)
	}

	seenTopics := map[string]bool{}
	for _, phase := range rm.Phases {
		for _, topic := range phase.Topics {
			if seenTopics[topic.Slug] {
				t.Errorf("duplicate topic slug: %q", topic.Slug)
			}
			seenTopics[topic.Slug] = true
		}
	}
}
