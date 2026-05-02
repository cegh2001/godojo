package domain

import "fmt"

// Exercise represents a complete exercise with template, test, and solution code.
type Exercise struct {
	Slug         string
	Title        string
	TopicSlug    string
	TemplateCode string // skeleton ejercicio.go
	TestCode     string // ejercicio_test.go content
	SolutionCode string // optional reference solution
}

// NewExercise creates a validated Exercise.
func NewExercise(slug, title, topicSlug, templateCode, testCode, solutionCode string) (*Exercise, error) {
	if slug == "" {
		return nil, fmt.Errorf("exercise slug is required")
	}
	if title == "" {
		return nil, fmt.Errorf("exercise title is required")
	}
	if topicSlug == "" {
		return nil, fmt.Errorf("topic slug is required")
	}
	if templateCode == "" {
		return nil, fmt.Errorf("template code is required")
	}
	if testCode == "" {
		return nil, fmt.Errorf("test code is required")
	}
	return &Exercise{
		Slug:         slug,
		Title:        title,
		TopicSlug:    topicSlug,
		TemplateCode: templateCode,
		TestCode:     testCode,
		SolutionCode: solutionCode,
	}, nil
}
