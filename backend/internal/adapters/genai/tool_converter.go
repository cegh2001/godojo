package genai

import (
	"godojo/internal/core/domain"
	g "google.golang.org/genai"
)

// ToolDeclarationToGenai converts a domain ToolDeclaration to a genai FunctionDeclaration.
func ToolDeclarationToGenai(decl domain.ToolDeclaration) *g.FunctionDeclaration {
	return &g.FunctionDeclaration{
		Name:        decl.Name,
		Description: decl.Description,
		Parameters:  toolParametersToSchema(decl.Parameters),
	}
}

// ToolDeclarationsToGenai converts a slice of domain ToolDeclaration to genai FunctionDeclarations.
func ToolDeclarationsToGenai(decls []domain.ToolDeclaration) []*g.FunctionDeclaration {
	result := make([]*g.FunctionDeclaration, 0, len(decls))
	for i := range decls {
		result = append(result, ToolDeclarationToGenai(decls[i]))
	}
	return result
}

// toolParametersToSchema converts domain ToolParameters to a genai Schema pointer.
func toolParametersToSchema(params domain.ToolParameters) *g.Schema {
	properties := make(map[string]*g.Schema, len(params.Properties))
	for name, prop := range params.Properties {
		properties[name] = ToolPropertyToSchema(prop)
	}

	// If there are no properties and no required fields, a nil schema is acceptable
	if len(properties) == 0 {
		return nil
	}

	return &g.Schema{
		Type:       g.TypeObject,
		Properties: properties,
		Required:   params.Required,
	}
}

// domainTypeToGenai maps domain type strings to genai Type constants.
// Domain uses lowercase (string, integer, number, boolean, object, array).
// Genai uses uppercase (STRING, INTEGER, NUMBER, BOOLEAN, OBJECT, ARRAY).
func domainTypeToGenai(domainType string) g.Type {
	switch domainType {
	case "string":
		return g.TypeString
	case "integer":
		return g.TypeInteger
	case "number":
		return g.TypeNumber
	case "boolean":
		return g.TypeBoolean
	case "object":
		return g.TypeObject
	case "array":
		return g.TypeArray
	default:
		return g.Type(domainType)
	}
}

// ToolPropertyToSchema converts a domain ToolProperty to a genai Schema pointer.
func ToolPropertyToSchema(prop domain.ToolProperty) *g.Schema {
	return &g.Schema{
		Type:        domainTypeToGenai(prop.Type),
		Description: prop.Description,
	}
}

// PartToDomain converts a genai Part to a domain ContentPart.
func PartToDomain(part *g.Part) domain.ContentPart {
	if part == nil {
		return domain.ContentPart{}
	}

	result := domain.ContentPart{
		Text: part.Text,
	}

	if part.FunctionCall != nil {
		result.FunctionCall = &domain.FunctionCall{
			ID:   part.FunctionCall.ID,
			Name: part.FunctionCall.Name,
			Args: part.FunctionCall.Args,
		}
	}

	return result
}

// PartsToDomain converts a slice of genai Parts to domain ContentParts.
func PartsToDomain(parts []*g.Part) []domain.ContentPart {
	result := make([]domain.ContentPart, 0, len(parts))
	for _, part := range parts {
		result = append(result, PartToDomain(part))
	}
	return result
}
