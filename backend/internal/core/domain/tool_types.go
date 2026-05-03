package domain

// ToolDeclaration describes a tool that the sensei can invoke.
// Follows the Gemini function calling JSON schema.
type ToolDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
}

// ToolParameters defines the input schema for a tool.
type ToolParameters struct {
	Type       string                  `json:"type"`
	Properties map[string]ToolProperty `json:"properties"`
	Required   []string                `json:"required,omitempty"`
}

// ToolProperty describes a single parameter of a tool.
type ToolProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}
