package domain_test

import (
	"encoding/json"
	"testing"

	"godojo/internal/core/domain"
)

func TestContentPartJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.ContentPart
		wantKey string // key that must exist in JSON
	}{
		{
			name: "text-only — JSON has 'text' field",
			input: domain.ContentPart{
				Text: "Hola, ¿cómo estás?",
			},
			wantKey: "text",
		},
		{
			name: "functionCall only — JSON has 'functionCall' nested object",
			input: domain.ContentPart{
				FunctionCall: &domain.FunctionCall{
					Name: "create_exercise_file",
					Args: map[string]interface{}{
						"filename": "hola-mundo.go",
						"content":  "package main\n\nfunc main() {}",
					},
				},
			},
			wantKey: "functionCall",
		},
		{
			name: "both text and functionCall",
			input: domain.ContentPart{
				Text: "Voy a crear un archivo:",
				FunctionCall: &domain.FunctionCall{
					Name: "create_exercise_file",
					Args: map[string]interface{}{
						"filename": "variables.go",
					},
				},
			},
			wantKey: "functionCall",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Verify the key exists in JSON
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal into map: %v", err)
			}

			if _, ok := raw[tt.wantKey]; !ok {
				t.Errorf("JSON missing %q key. Got: %s", tt.wantKey, string(data))
			}

			// Round-trip: unmarshal back into ContentPart
			var roundTripped domain.ContentPart
			if err := json.Unmarshal(data, &roundTripped); err != nil {
				t.Fatalf("failed to unmarshal round-trip: %v", err)
			}

			// Verify text field
			if roundTripped.Text != tt.input.Text {
				t.Errorf("round-trip text = %q, want %q", roundTripped.Text, tt.input.Text)
			}

			// Verify functionCall field
			if tt.input.FunctionCall != nil {
				if roundTripped.FunctionCall == nil {
					t.Fatalf("round-trip lost FunctionCall")
				}
				if roundTripped.FunctionCall.Name != tt.input.FunctionCall.Name {
					t.Errorf("round-trip FunctionCall.Name = %q, want %q",
						roundTripped.FunctionCall.Name, tt.input.FunctionCall.Name)
				}
			}
		})
	}
}

func TestFunctionCallEmptyArgsRoundTrip(t *testing.T) {
	// FunctionCall with empty (but non-nil) Args must survive round-trip
	fc := domain.FunctionCall{
		Name: "read_roadmap_section",
		Args: map[string]interface{}{},
	}

	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var roundTripped domain.FunctionCall
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if roundTripped.Name != fc.Name {
		t.Errorf("Name = %q, want %q", roundTripped.Name, fc.Name)
	}

	if len(roundTripped.Args) != 0 {
		t.Errorf("Args len = %d, want 0", len(roundTripped.Args))
	}
}

func TestFunctionCallNilArgsRoundTrip(t *testing.T) {
	// FunctionCall with nil Args must survive round-trip
	fc := domain.FunctionCall{
		Name: "no_args_tool",
		Args: nil,
	}

	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var roundTripped domain.FunctionCall
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if roundTripped.Name != fc.Name {
		t.Errorf("Name = %q, want %q", roundTripped.Name, fc.Name)
	}

	if roundTripped.Args != nil {
		t.Errorf("Args should be nil, got %v", roundTripped.Args)
	}
}

func TestContentPartOmitempty(t *testing.T) {
	tests := []struct {
		name      string
		cp        domain.ContentPart
		assertKey string // must exist
		assertNot string // must NOT exist
	}{
		{
			name:      "text only omits functionCall",
			cp:        domain.ContentPart{Text: "solo texto"},
			assertKey: "text",
			assertNot: "functionCall",
		},
		{
			name:      "functionCall only omits text",
			cp:        domain.ContentPart{FunctionCall: &domain.FunctionCall{Name: "f", Args: map[string]interface{}{"a": 1}}},
			assertKey: "functionCall",
			assertNot: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.cp)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal into map: %v", err)
			}

			// Verify expected key exists
			if _, ok := raw[tt.assertKey]; !ok {
				t.Errorf("JSON missing %q key. Got: %s", tt.assertKey, string(data))
			}

			// Verify unexpected key does NOT exist
			if _, ok := raw[tt.assertNot]; ok {
				t.Errorf("JSON should NOT have %q key. Got: %s", tt.assertNot, string(data))
			}
		})
	}
}
