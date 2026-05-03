package domain_test

import (
	"encoding/json"
	"testing"

	"godojo/internal/core/domain"
)

func TestToolDeclarationJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input domain.ToolDeclaration
	}{
		{
			name: "full declaration with required fields",
			input: domain.ToolDeclaration{
				Name:        "create_exercise_file",
				Description: "Crea un archivo .go en el workspace del estudiante",
				Parameters: domain.ToolParameters{
					Type: "object",
					Properties: map[string]domain.ToolProperty{
						"filename": {Type: "string", Description: "Nombre del archivo .go"},
						"content":  {Type: "string", Description: "Contenido completo del archivo"},
					},
					Required: []string{"filename", "content"},
				},
			},
		},
		{
			name: "declaration without required fields",
			input: domain.ToolDeclaration{
				Name:        "read_roadmap_section",
				Description: "Lee una sección del roadmap",
				Parameters: domain.ToolParameters{
					Type: "object",
					Properties: map[string]domain.ToolProperty{
						"section_slug": {Type: "string", Description: "Slug de la sección a leer"},
					},
					Required: nil,
				},
			},
		},
		{
			name: "declaration with empty properties and no required",
			input: domain.ToolDeclaration{
				Name:        "ping",
				Description: "Simple ping tool",
				Parameters: domain.ToolParameters{
					Type:       "object",
					Properties: map[string]domain.ToolProperty{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Verify structure: name and description must exist
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal into map: %v", err)
			}

			if name, ok := raw["name"].(string); !ok || name != tt.input.Name {
				t.Errorf("JSON name = %v, want %q", raw["name"], tt.input.Name)
			}
			if desc, ok := raw["description"].(string); !ok || desc != tt.input.Description {
				t.Errorf("JSON description = %v, want %q", raw["description"], tt.input.Description)
			}

			// Verify parameters structure
			params, ok := raw["parameters"].(map[string]interface{})
			if !ok {
				t.Fatalf("parameters not a map: %T", raw["parameters"])
			}
			if paramsType, ok := params["type"].(string); !ok || paramsType != "object" {
				t.Errorf("parameters.type = %v, want 'object'", params["type"])
			}

			// Round-trip back to struct
			var roundTripped domain.ToolDeclaration
			if err := json.Unmarshal(data, &roundTripped); err != nil {
				t.Fatalf("failed to unmarshal round-trip: %v", err)
			}

			if roundTripped.Name != tt.input.Name {
				t.Errorf("round-trip name = %q, want %q", roundTripped.Name, tt.input.Name)
			}
			if roundTripped.Description != tt.input.Description {
				t.Errorf("round-trip description = %q, want %q", roundTripped.Description, tt.input.Description)
			}
			if len(roundTripped.Parameters.Properties) != len(tt.input.Parameters.Properties) {
				t.Errorf("round-trip properties len = %d, want %d",
					len(roundTripped.Parameters.Properties), len(tt.input.Parameters.Properties))
			}
			if len(roundTripped.Parameters.Required) != len(tt.input.Parameters.Required) {
				t.Errorf("round-trip required len = %d, want %d",
					len(roundTripped.Parameters.Required), len(tt.input.Parameters.Required))
			}
		})
	}
}

func TestToolParametersRequiredOmitempty(t *testing.T) {
	// When Required is empty, JSON must NOT include "required" key
	params := domain.ToolParameters{
		Type: "object",
		Properties: map[string]domain.ToolProperty{
			"query": {Type: "string", Description: "Búsqueda a realizar"},
		},
		Required: []string{}, // empty slice, not nil
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, exists := raw["required"]; exists {
		t.Errorf("JSON should NOT have 'required' key when empty. Got: %s", string(data))
	}
}

func TestToolParametersRequiredNilOmitempty(t *testing.T) {
	// When Required is nil, JSON must NOT include "required" key
	params := domain.ToolParameters{
		Type: "object",
		Properties: map[string]domain.ToolProperty{
			"query": {Type: "string", Description: "Búsqueda a realizar"},
		},
		Required: nil,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, exists := raw["required"]; exists {
		t.Errorf("JSON should NOT have 'required' key when nil. Got: %s", string(data))
	}
}

func TestToolDeclarationGeminiFormat(t *testing.T) {
	// Verify the JSON matches Gemini function calling expected format
	td := domain.ToolDeclaration{
		Name:        "googleSearch",
		Description: "Busca información en la web",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"query": {Type: "string", Description: "Término de búsqueda"},
			},
			Required: []string{"query"},
		},
	}

	data, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify it can be parsed as a function declaration in Gemini's expected format
	var result struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  struct {
			Type       string                 `json:"type"`
			Properties map[string]interface{} `json:"properties"`
			Required   []string               `json:"required"`
		} `json:"parameters"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal into Gemini format struct: %v", err)
	}

	if result.Name != "googleSearch" {
		t.Errorf("name = %q, want 'googleSearch'", result.Name)
	}
	if result.Parameters.Type != "object" {
		t.Errorf("parameters.type = %q, want 'object'", result.Parameters.Type)
	}
	if _, ok := result.Parameters.Properties["query"]; !ok {
		t.Errorf("properties missing 'query' key")
	}
	if len(result.Parameters.Required) != 1 || result.Parameters.Required[0] != "query" {
		t.Errorf("required = %v, want ['query']", result.Parameters.Required)
	}
}
