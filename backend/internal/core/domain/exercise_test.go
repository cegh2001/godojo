package domain_test

import (
	"testing"

	"godojo/internal/core/domain"
)

func TestNewExercise(t *testing.T) {
	tests := []struct {
		name         string
		slug         string
		title        string
		topicSlug    string
		templateCode string
		testCode     string
		solutionCode string
		wantErr      bool
	}{
		{
			name:         "valid exercise with all fields",
			slug:         "variables-basicas",
			title:        "Variables Básicas",
			topicSlug:    "variables",
			templateCode: "package main\n\nfunc main() {}",
			testCode:     "package main\n\nimport \"testing\"\n\nfunc TestDummy(t *testing.T) {}",
			solutionCode: "// Solución de ejemplo",
			wantErr:      false,
		},
		{
			name:         "valid exercise without solution code",
			slug:         "funciones",
			title:        "Funciones",
			topicSlug:    "funciones",
			templateCode: "package main",
			testCode:     "package main",
			solutionCode: "",
			wantErr:      false,
		},
		{
			name:         "empty slug",
			slug:         "",
			title:        "Título",
			topicSlug:    "topic",
			templateCode: "code",
			testCode:     "code",
			wantErr:      true,
		},
		{
			name:         "empty title",
			slug:         "slug",
			title:        "",
			topicSlug:    "topic",
			templateCode: "code",
			testCode:     "code",
			wantErr:      true,
		},
		{
			name:         "empty topic slug",
			slug:         "slug",
			title:        "Título",
			topicSlug:    "",
			templateCode: "code",
			testCode:     "code",
			wantErr:      true,
		},
		{
			name:         "empty template code",
			slug:         "slug",
			title:        "Título",
			topicSlug:    "topic",
			templateCode: "",
			testCode:     "code",
			wantErr:      true,
		},
		{
			name:         "empty test code",
			slug:         "slug",
			title:        "Título",
			topicSlug:    "topic",
			templateCode: "code",
			testCode:     "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex, err := domain.NewExercise(tt.slug, tt.title, tt.topicSlug, tt.templateCode, tt.testCode, tt.solutionCode)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil, ex=%+v", ex)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if ex.Slug != tt.slug {
				t.Errorf("slug = %q, want %q", ex.Slug, tt.slug)
			}
			if ex.Title != tt.title {
				t.Errorf("title = %q, want %q", ex.Title, tt.title)
			}
			if ex.TopicSlug != tt.topicSlug {
				t.Errorf("topicSlug = %q, want %q", ex.TopicSlug, tt.topicSlug)
			}
			if ex.TemplateCode != tt.templateCode {
				t.Errorf("templateCode mismatch")
			}
			if ex.TestCode != tt.testCode {
				t.Errorf("testCode mismatch")
			}
			if ex.SolutionCode != tt.solutionCode {
				t.Errorf("solutionCode mismatch")
			}
		})
	}
}
