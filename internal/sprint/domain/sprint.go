package domain

import "strings"

// Sprint represents a project iteration and its goal.
type Sprint struct {
	ID         string
	ProjectID  string
	SprintGoal string
	IsClosed   bool
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
