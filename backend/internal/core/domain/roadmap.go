package domain

import (
	"fmt"
)

// Difficulty represents the difficulty level of an exercise.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// validDifficulties is used for validation.
var validDifficulties = map[Difficulty]bool{
	DifficultyEasy:   true,
	DifficultyMedium: true,
	DifficultyHard:   true,
}

// ExerciseRef is a lightweight reference to an exercise within a topic.
type ExerciseRef struct {
	Slug       string
	Title      string
	Difficulty Difficulty
}

// NewExerciseRef creates a validated ExerciseRef.
func NewExerciseRef(slug, title, difficulty string) (*ExerciseRef, error) {
	if slug == "" {
		return nil, fmt.Errorf("exercise slug is required")
	}
	if title == "" {
		return nil, fmt.Errorf("exercise title is required")
	}
	d := Difficulty(difficulty)
	if !validDifficulties[d] {
		return nil, fmt.Errorf("invalid difficulty %q: must be easy, medium, or hard", difficulty)
	}
	return &ExerciseRef{
		Slug:       slug,
		Title:      title,
		Difficulty: d,
	}, nil
}

// Phase represents a learning phase with a collection of topics.
type Phase struct {
	ID     string
	Title  string
	Topics []*Topic
}

// NewPhase creates a validated Phase.
func NewPhase(id, title string, topics []*Topic) (*Phase, error) {
	if id == "" {
		return nil, fmt.Errorf("phase id is required")
	}
	if title == "" {
		return nil, fmt.Errorf("phase title is required")
	}
	if topics == nil {
		topics = []*Topic{}
	}
	return &Phase{
		ID:     id,
		Title:  title,
		Topics: topics,
	}, nil
}

// Topic represents a learning topic containing exercises.
type Topic struct {
	Slug         string
	Title        string
	Description  string
	Exercises    []*ExerciseRef
	Dependencies []string // topic slugs required before unlocking
}

// NewTopic creates a validated Topic.
func NewTopic(slug, title, description string, exercises []*ExerciseRef, dependencies []string) (*Topic, error) {
	if slug == "" {
		return nil, fmt.Errorf("topic slug is required")
	}
	if title == "" {
		return nil, fmt.Errorf("topic title is required")
	}
	if exercises == nil {
		exercises = []*ExerciseRef{}
	}
	if dependencies == nil {
		dependencies = []string{}
	}
	return &Topic{
		Slug:         slug,
		Title:        title,
		Description:  description,
		Exercises:    exercises,
		Dependencies: dependencies,
	}, nil
}

// Roadmap is the full learning path containing phases.
type Roadmap struct {
	Phases []*Phase
}

// NewRoadmap creates a validated Roadmap.
func NewRoadmap(phases []*Phase) (*Roadmap, error) {
	if len(phases) == 0 {
		return nil, fmt.Errorf("roadmap must have at least one phase")
	}
	return &Roadmap{
		Phases: phases,
	}, nil
}
