package genai

import (
	"testing"

	"godojo/internal/core/domain"
	g "google.golang.org/genai"
)

func TestToolDeclarationToGenai_SingleTool(t *testing.T) {
	decl := domain.ToolDeclaration{
		Name:        "create_exercise_file",
		Description: "Crea un archivo de ejercicio Go en el workspace",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "string", Description: "Nombre del archivo .go"},
				"content":  {Type: "string", Description: "Contenido del archivo"},
			},
			Required: []string{"filename"},
		},
	}

	result := ToolDeclarationToGenai(decl)

	if result.Name != "create_exercise_file" {
		t.Errorf("Name = %q, want %q", result.Name, "create_exercise_file")
	}
	if result.Description != "Crea un archivo de ejercicio Go en el workspace" {
		t.Errorf("Description mismatch")
	}
	if result.Parameters == nil {
		t.Fatal("Parameters should not be nil")
	}
	if result.Parameters.Type != g.TypeObject {
		t.Errorf("Parameters.Type = %q, want %q", result.Parameters.Type, g.TypeObject)
	}
	if len(result.Parameters.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(result.Parameters.Properties))
	}
	if len(result.Parameters.Required) != 1 || result.Parameters.Required[0] != "filename" {
		t.Errorf("Required = %v, want [filename]", result.Parameters.Required)
	}

	// Verify individual properties
	filenameProp, ok := result.Parameters.Properties["filename"]
	if !ok {
		t.Fatal("missing 'filename' property")
	}
	if filenameProp.Type != g.TypeString {
		t.Errorf("filename.Type = %q, want %q", filenameProp.Type, g.TypeString)
	}
	if filenameProp.Description != "Nombre del archivo .go" {
		t.Errorf("filename.Description mismatch: %q", filenameProp.Description)
	}

	contentProp, ok := result.Parameters.Properties["content"]
	if !ok {
		t.Fatal("missing 'content' property")
	}
	if contentProp.Type != g.TypeString {
		t.Errorf("content.Type = %q, want %q", contentProp.Type, g.TypeString)
	}
}

func TestToolDeclarationToGenai_NoParameters(t *testing.T) {
	decl := domain.ToolDeclaration{
		Name:        "simple_tool",
		Description: "A tool with no parameters",
		Parameters: domain.ToolParameters{
			Type: "object",
		},
	}

	result := ToolDeclarationToGenai(decl)

	if result.Parameters != nil {
		// When Properties is empty, it's acceptable to have Parameters or nil
		if len(result.Parameters.Properties) > 0 {
			t.Errorf("expected no properties, got %d", len(result.Parameters.Properties))
		}
	}
}

func TestToolDeclarationsToGenai_MultipleTools(t *testing.T) {
	decls := []domain.ToolDeclaration{
		{
			Name:        "tool_a",
			Description: "First tool",
			Parameters: domain.ToolParameters{
				Type:       "object",
				Properties: map[string]domain.ToolProperty{},
			},
		},
		{
			Name:        "tool_b",
			Description: "Second tool",
			Parameters: domain.ToolParameters{
				Type:       "object",
				Properties: map[string]domain.ToolProperty{},
			},
		},
	}

	results := ToolDeclarationsToGenai(decls)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "tool_a" {
		t.Errorf("results[0].Name = %q, want tool_a", results[0].Name)
	}
	if results[1].Name != "tool_b" {
		t.Errorf("results[1].Name = %q, want tool_b", results[1].Name)
	}
}

func TestToolDeclarationsToGenai_EmptySlice(t *testing.T) {
	results := ToolDeclarationsToGenai([]domain.ToolDeclaration{})
	if results == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestPartToDomain_TextPart(t *testing.T) {
	part := &g.Part{
		Text: "Hola, ¿cómo estás?",
	}

	result := PartToDomain(part)

	if result.Text != "Hola, ¿cómo estás?" {
		t.Errorf("Text = %q, want %q", result.Text, "Hola, ¿cómo estás?")
	}
	if result.FunctionCall != nil {
		t.Error("FunctionCall should be nil for text-only part")
	}
}

func TestPartToDomain_FunctionCallPart(t *testing.T) {
	part := &g.Part{
		FunctionCall: &g.FunctionCall{
			ID:   "call-123",
			Name: "create_exercise_file",
			Args: map[string]any{
				"filename": "test.go",
				"content":  "package main",
			},
		},
	}

	result := PartToDomain(part)

	if result.FunctionCall == nil {
		t.Fatal("FunctionCall should not be nil")
	}
	if result.FunctionCall.ID != "call-123" {
		t.Errorf("FunctionCall.ID = %q, want %q", result.FunctionCall.ID, "call-123")
	}
	if result.FunctionCall.Name != "create_exercise_file" {
		t.Errorf("FunctionCall.Name = %q, want %q", result.FunctionCall.Name, "create_exercise_file")
	}
	if result.FunctionCall.Args["filename"] != "test.go" {
		t.Errorf("Args[filename] = %q, want test.go", result.FunctionCall.Args["filename"])
	}
	if result.FunctionCall.Args["content"] != "package main" {
		t.Errorf("Args[content] = %q, want package main", result.FunctionCall.Args["content"])
	}
	if result.Text != "" {
		t.Errorf("Text should be empty for function-call-only part, got %q", result.Text)
	}
}

func TestPartToDomain_NilPart(t *testing.T) {
	result := PartToDomain(nil)
	// Should handle nil gracefully
	if result.Text != "" || result.FunctionCall != nil {
		t.Error("nil part should produce empty ContentPart")
	}
}

func TestPartsToDomain_MultipleParts(t *testing.T) {
	parts := []*g.Part{
		{Text: "Vamos a crear un archivo."},
		{
			FunctionCall: &g.FunctionCall{
				ID:   "call-456",
				Name: "read_codebase",
				Args: map[string]any{"topic_slug": "variables"},
			},
		},
	}

	results := PartsToDomain(parts)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Text != "Vamos a crear un archivo." {
		t.Errorf("results[0].Text = %q", results[0].Text)
	}
	if results[1].FunctionCall == nil {
		t.Fatal("results[1].FunctionCall should not be nil")
	}
	if results[1].FunctionCall.Name != "read_codebase" {
		t.Errorf("results[1].FunctionCall.Name = %q", results[1].FunctionCall.Name)
	}
}

func TestPartsToDomain_EmptySlice(t *testing.T) {
	results := PartsToDomain([]*g.Part{})
	if results == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestToolPropertyToSchema_TypeMapping(t *testing.T) {
	tests := []struct {
		name         string
		propertyType string
		want         g.Type
	}{
		{"string type", "string", g.TypeString},
		{"integer type", "integer", g.TypeInteger},
		{"number type", "number", g.TypeNumber},
		{"boolean type", "boolean", g.TypeBoolean},
		{"object type", "object", g.TypeObject},
		{"array type", "array", g.TypeArray},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop := domain.ToolProperty{
				Type:        tt.propertyType,
				Description: "test property",
			}
			schema := ToolPropertyToSchema(prop)

			if schema.Type != tt.want {
				t.Errorf("Type = %q, want %q", schema.Type, tt.want)
			}
			if schema.Description != "test property" {
				t.Errorf("Description = %q, want test property", schema.Description)
			}
		})
	}
}

func TestToolPropertyToSchema_UnknownType(t *testing.T) {
	prop := domain.ToolProperty{
		Type:        "custom_type",
		Description: "Some custom type",
	}

	schema := ToolPropertyToSchema(prop)

	if schema.Type != g.Type("custom_type") {
		t.Errorf("unknown type should fall through as raw string, got %q", schema.Type)
	}
	if schema.Description != "Some custom type" {
		t.Errorf("Description = %q, want 'Some custom type'", schema.Description)
	}
}

func TestPartToDomain_TextAndFunctionCall(t *testing.T) {
	// Part can have both Text and FunctionCall (multi-tool calls in same turn)
	part := &g.Part{
		Text: "Voy a ayudarte con eso.",
		FunctionCall: &g.FunctionCall{
			ID:   "call-789",
			Name: "setup_workspace",
			Args: map[string]any{"topic_slug": "variables"},
		},
	}

	result := PartToDomain(part)

	if result.Text != "Voy a ayudarte con eso." {
		t.Errorf("Text = %q", result.Text)
	}
	if result.FunctionCall == nil {
		t.Fatal("FunctionCall should not be nil")
	}
	if result.FunctionCall.Name != "setup_workspace" {
		t.Errorf("FunctionCall.Name = %q", result.FunctionCall.Name)
	}
}

func TestToolDeclarationToGenai_PropertyTypes(t *testing.T) {
	decl := domain.ToolDeclaration{
		Name:        "typed_tool",
		Description: "Tool with various property types",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"count":    {Type: "integer", Description: "Number of items"},
				"enabled":  {Type: "boolean", Description: "Whether enabled"},
				"ratio":    {Type: "number", Description: "A float value"},
				"tags":     {Type: "array", Description: "List of tags"},
				"metadata": {Type: "object", Description: "Extra metadata"},
			},
			Required: []string{"count"},
		},
	}

	result := ToolDeclarationToGenai(decl)

	if result.Parameters == nil {
		t.Fatal("Parameters should not be nil")
	}

	typeChecks := map[string]g.Type{
		"count":    g.TypeInteger,
		"enabled":  g.TypeBoolean,
		"ratio":    g.TypeNumber,
		"tags":     g.TypeArray,
		"metadata": g.TypeObject,
	}

	for propName, expectedType := range typeChecks {
		prop, ok := result.Parameters.Properties[propName]
		if !ok {
			t.Errorf("missing property %q", propName)
			continue
		}
		if prop.Type != expectedType {
			t.Errorf("%s.Type = %q, want %q", propName, prop.Type, expectedType)
		}
	}
}
