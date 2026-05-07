package domain

// ContentPart represents a piece of a sensei response.
// It can be either plain text or a function call (or both).
type ContentPart struct {
	Text         string        `json:"text,omitempty"`
	FunctionCall *FunctionCall `json:"functionCall,omitempty"`
}

// FunctionCall represents a tool invocation requested by the sensei.
type FunctionCall struct {
	ID   string                 `json:"id,omitempty"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}
