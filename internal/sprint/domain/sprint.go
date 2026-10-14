package domain

import (
	"strings"
	"time"
)

// Sprint represents a project iteration and its goal.
type Sprint struct {
	ID         string
	ProjectID  string
	SprintGoal string
}

// ValidationError describes invalid sprint input by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "sprint validation failed"
}

// NewSprint validates the required goal and preserves the supplied values.
func NewSprint(id, projectID, sprintGoal string) (Sprint, error) {
	if strings.TrimSpace(sprintGoal) == "" {
		return Sprint{}, &ValidationError{Fields: map[string]string{"sprint_goal": "is required"}}
	}

	return Sprint{ID: id, ProjectID: projectID, SprintGoal: sprintGoal}, nil
}

// ClosedSprint is a read-only snapshot of a closed sprint and the stories planned in it.
type ClosedSprint struct {
	ID         string
	ProjectID  string
	SprintGoal string
	Stories    []PlannedStory
}

// PlannedStory is a story assigned to a sprint, with its completion in that sprint.
type PlannedStory struct {
	ID          string
	Title       string
	Priority    string
	StoryPoints *int
	CompletedAt *time.Time
}

// Completed reports whether the story was completed in the sprint that contains it.
func (s PlannedStory) Completed() bool {
	return s.CompletedAt != nil
}

// SprintHistory is the history of a project's closed sprints, from the most recent to the oldest.
type SprintHistory struct {
	ProjectID string
	Sprints   []ClosedSprint
}
