package core_test

import (
	"errors"
	"sync"
	"testing"

	"godojo/internal/core"
	"godojo/internal/core/domain"
)

func TestToolRegistry_RegisterAndExecute(t *testing.T) {
	registry := core.NewToolRegistry()

	handler := func(args map[string]interface{}) (interface{}, error) {
		return args["name"].(string) + " dice hola", nil
	}

	registry.Register("saludar", domain.ToolDeclaration{
		Name:        "saludar",
		Description: "Saluda a alguien",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"name": {Type: "string", Description: "Nombre de la persona"},
			},
			Required: []string{"name"},
		},
	}, handler)

	result, err := registry.Execute("saludar", map[string]interface{}{
		"name": "Juan",
	})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result != "Juan dice hola" {
		t.Errorf("result = %v, want 'Juan dice hola'", result)
	}
}

func TestToolRegistry_ExecuteUnknownTool(t *testing.T) {
	registry := core.NewToolRegistry()

	_, err := registry.Execute("herramienta_inexistente", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}
}

func TestToolRegistry_ExecuteMissingRequiredArg(t *testing.T) {
	registry := core.NewToolRegistry()

	registry.Register("crear_archivo", domain.ToolDeclaration{
		Name:        "crear_archivo",
		Description: "Crea un archivo",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "string", Description: "Nombre del archivo"},
				"content":  {Type: "string", Description: "Contenido"},
			},
			Required: []string{"filename", "content"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return "ok", nil
	})

	// Missing "content" argument
	_, err := registry.Execute("crear_archivo", map[string]interface{}{
		"filename": "test.go",
	})
	if err == nil {
		t.Fatal("expected error for missing required arg, got nil")
	}
}

func TestToolRegistry_ExecuteHandlerError(t *testing.T) {
	registry := core.NewToolRegistry()

	registry.Register("rompe", domain.ToolDeclaration{
		Name:        "rompe",
		Description: "Siempre falla",
		Parameters: domain.ToolParameters{
			Type:       "object",
			Properties: map[string]domain.ToolProperty{},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return nil, errors.New("error forzado del handler")
	})

	_, err := registry.Execute("rompe", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error from handler, got nil")
	}
}

func TestToolRegistry_GetDeclarations(t *testing.T) {
	registry := core.NewToolRegistry()

	registry.Register("tool_a", domain.ToolDeclaration{
		Name:        "tool_a",
		Description: "Primera herramienta",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"param1": {Type: "string", Description: "Param 1"},
			},
			Required: []string{"param1"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return nil, nil
	})

	registry.Register("tool_b", domain.ToolDeclaration{
		Name:        "tool_b",
		Description: "Segunda herramienta",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"param2": {Type: "number", Description: "Param 2"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return nil, nil
	})

	tools := registry.GetDeclarations()
	if len(tools) != 2 {
		t.Fatalf("len(declarations) = %d, want 2", len(tools))
	}

	// Verify both tools are present
	names := make(map[string]bool)
	for _, td := range tools {
		names[td.Name] = true
	}
	if !names["tool_a"] {
		t.Error("missing tool_a in declarations")
	}
	if !names["tool_b"] {
		t.Error("missing tool_b in declarations")
	}
}

func TestToolRegistry_ExecuteNoRequiredArgs(t *testing.T) {
	registry := core.NewToolRegistry()

	registry.Register("ping", domain.ToolDeclaration{
		Name:        "ping",
		Description: "Simple ping sin parámetros",
		Parameters: domain.ToolParameters{
			Type:       "object",
			Properties: map[string]domain.ToolProperty{},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return "pong", nil
	})

	result, err := registry.Execute("ping", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result != "pong" {
		t.Errorf("result = %v, want 'pong'", result)
	}
}

func TestToolRegistry_ExecuteExtraArgsIgnored(t *testing.T) {
	// Extra args not in declaration should be ignored (passed to handler)
	registry := core.NewToolRegistry()

	registry.Register("eco", domain.ToolDeclaration{
		Name:        "eco",
		Description: "Devuelve el mensaje",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"msg": {Type: "string", Description: "Mensaje"},
			},
			Required: []string{"msg"},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return args["msg"], nil
	})

	// Send extra args that handler can use but aren't declared
	result, err := registry.Execute("eco", map[string]interface{}{
		"msg":     "hola",
		"unknown": 42,
	})
	if err != nil {
		t.Fatalf("Execute with extra args failed: %v", err)
	}
	if result != "hola" {
		t.Errorf("result = %v, want 'hola'", result)
	}
}

func TestToolRegistry_GetDeclarationsPreservesFields(t *testing.T) {
	registry := core.NewToolRegistry()

	decl := domain.ToolDeclaration{
		Name:        "create_exercise_file",
		Description: "Crea un archivo .go en el workspace",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"filename": {Type: "string", Description: "Nombre del archivo .go"},
				"content":  {Type: "string", Description: "Contenido completo del archivo"},
			},
			Required: []string{"filename", "content"},
		},
	}

	registry.Register(decl.Name, decl, func(args map[string]interface{}) (interface{}, error) {
		return "ok", nil
	})

	tools := registry.GetDeclarations()
	if len(tools) != 1 {
		t.Fatalf("len(declarations) = %d, want 1", len(tools))
	}

	got := tools[0]
	if got.Name != decl.Name {
		t.Errorf("Name = %q, want %q", got.Name, decl.Name)
	}
	if got.Description != decl.Description {
		t.Errorf("Description = %q, want %q", got.Description, decl.Description)
	}
	if got.Parameters.Type != decl.Parameters.Type {
		t.Errorf("Parameters.Type = %q, want %q", got.Parameters.Type, decl.Parameters.Type)
	}
	if len(got.Parameters.Properties) != len(decl.Parameters.Properties) {
		t.Errorf("Properties len = %d, want %d",
			len(got.Parameters.Properties), len(decl.Parameters.Properties))
	}
	if len(got.Parameters.Required) != len(decl.Parameters.Required) {
		t.Errorf("Required len = %d, want %d",
			len(got.Parameters.Required), len(decl.Parameters.Required))
	}
}

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	registry := core.NewToolRegistry()

	// Pre-register a tool
	registry.Register("tool", domain.ToolDeclaration{
		Name:        "tool",
		Description: "Test tool",
		Parameters: domain.ToolParameters{
			Type: "object",
			Properties: map[string]domain.ToolProperty{
				"x": {Type: "integer", Description: "Un número"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		return args["x"], nil
	})

	var wg sync.WaitGroup

	// Concurrent executes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			result, err := registry.Execute("tool", map[string]interface{}{"x": val})
			if err != nil {
				t.Errorf("concurrent Execute failed: %v", err)
				return
			}
			// result from JSON unmarshaling might be float64, handle both
			switch v := result.(type) {
			case int:
				if v != val {
					t.Errorf("result = %d, want %d", v, val)
				}
			case float64:
				if int(v) != val {
					t.Errorf("result = %f, want %d", v, val)
				}
			default:
				t.Errorf("unexpected result type: %T", result)
			}
		}(i)
	}

	// Concurrent registers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(suffix int) {
			defer wg.Done()
			name := "tool_" + string(rune('a'+suffix))
			registry.Register(name, domain.ToolDeclaration{
				Name:        name,
				Description: "Concurrent tool",
				Parameters: domain.ToolParameters{
					Type: "object",
					Properties: map[string]domain.ToolProperty{
						"val": {Type: "integer", Description: "Valor"},
					},
				},
			}, func(args map[string]interface{}) (interface{}, error) {
				return "ok", nil
			})
		}(i)
	}

	// Concurrent GetDeclarations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.GetDeclarations()
		}()
	}

	wg.Wait()

	// After all concurrent ops, verify we can still use the registry
	tools := registry.GetDeclarations()
	if len(tools) < 1 {
		t.Errorf("should have at least 1 tool, got %d", len(tools))
	}
}
