package domain

import (
	"strings"
	"time"
)

// Project contains the basic data required to create a project.
type Project struct {
	ID                string
	Name              string
	StartDate         time.Time
	PlannedFinishDate time.Time
}

// ValidationError describes invalid input by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "project validation failed"
}

// NewProject creates a project that satisfies the basic project invariants.
func NewProject(id, name string, startDate, plannedFinishDate time.Time) (Project, error) {
	fields := make(map[string]string)

	if strings.TrimSpace(name) == "" {
		fields["name"] = "is required"
	}
	if startDate.IsZero() {
		fields["start_date"] = "is required"
	}
	if plannedFinishDate.IsZero() {
		fields["planned_finish_date"] = "is required"
	}
	if !startDate.IsZero() && !plannedFinishDate.IsZero() && plannedFinishDate.Before(startDate) {
		fields["planned_finish_date"] = "must be on or after start_date"
	}
	if len(fields) > 0 {
		return Project{}, &ValidationError{Fields: fields}
	}

	return Project{
		ID:                id,
		Name:              name,
		StartDate:         startDate,
		PlannedFinishDate: plannedFinishDate,
	}, nil
}
