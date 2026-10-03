// Package domain contains the Task entity and its validation rules.
package domain

import (
	"strconv"
	"strings"
)

const (
	maxEstimatedHours        = 99999.99
	maxEstimatedHoursDecimal = 2
	estimatedHoursMessage    = "must be greater than 0, at most 99999.99 and have at most 2 decimals"
)

// Task is a unit of work into which a story assigned to a sprint is decomposed.
type Task struct {
	ID             string
	ProjectID      string
	SprintID       string
	StoryID        string
	Title          string
	EstimatedHours *float64 // nil means "not estimated"
}

// ValidationError describes invalid task input by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "task validation failed" }

// NewTask validates title (required, non-blank after TrimSpace, stored untrimmed) and the
// optional estimate (> 0, <= 99999.99, at most 2 decimals). On failure it returns
// *ValidationError with keys "title" and/or "estimated_hours". The returned task holds a
// copy of the estimate, never the caller's pointer.
func NewTask(id, projectID, sprintID, storyID, title string, estimatedHours *float64) (Task, error) {
	fields := make(map[string]string)
	if strings.TrimSpace(title) == "" {
		fields["title"] = "is required"
	}
	validateEstimatedHours(fields, estimatedHours)
	if len(fields) > 0 {
		return Task{}, &ValidationError{Fields: fields}
	}

	var hours *float64
	if estimatedHours != nil {
		value := *estimatedHours
		hours = &value
	}
	return Task{
		ID:             id,
		ProjectID:      projectID,
		SprintID:       sprintID,
		StoryID:        storyID,
		Title:          title,
		EstimatedHours: hours,
	}, nil
}

// validateEstimatedHours accumulates in fields the estimate rules: greater than 0,
// at most 99999.99 and at most 2 decimals. Decimals are counted on the shortest
// decimal representation that round-trips the float64, not with float arithmetic.
func validateEstimatedHours(fields map[string]string, estimatedHours *float64) {
	if estimatedHours == nil {
		return
	}
	value := *estimatedHours
	if !(value > 0) || value > maxEstimatedHours || decimalPlaces(value) > maxEstimatedHoursDecimal {
		fields["estimated_hours"] = estimatedHoursMessage
	}
}

func decimalPlaces(value float64) int {
	_, fraction, found := strings.Cut(strconv.FormatFloat(value, 'f', -1, 64), ".")
	if !found {
		return 0
	}
	return len(fraction)
}
