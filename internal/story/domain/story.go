package domain

import (
	"strconv"
	"strings"
)

const (
	// StatusPending is the initial status of every story.
	StatusPending = "pendiente"
	// StatusInProgress marks a story whose work has started.
	StatusInProgress = "en_progreso"
	// StatusCompleted marks a story whose work is done.
	StatusCompleted = "completada"
)

const (
	maxEstimatedHours        = 99999.99
	maxEstimatedHoursDecimal = 2
	estimatedHoursMessage    = "must be greater than 0, at most 99999.99 and have at most 2 decimals"
)

// AllowedStatuses returns a copy of the closed set of statuses, in order.
func AllowedStatuses() []string {
	return []string{StatusPending, StatusInProgress, StatusCompleted}
}

// Story is work registered in a project's backlog before estimation.
type Story struct {
	ID                 string
	ProjectID          string
	Title              string
	Description        string
	Priority           string
	Status             string
	StoryPoints        *int
	AcceptanceCriteria []string
	EstimatedHours     *float64
}

// ValidationError describes invalid story input by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "story validation failed"
}

// NewStory validates the supplied data and initializes a pending, unestimated story.
func NewStory(id, projectID, title, description, priority string, criteria []string) (Story, error) {
	fields := make(map[string]string)
	validateStoryContent(fields, title, description, priority, criteria)
	if len(fields) > 0 {
		return Story{}, &ValidationError{Fields: fields}
	}

	return Story{
		ID:                 id,
		ProjectID:          projectID,
		Title:              title,
		Description:        description,
		Priority:           priority,
		Status:             StatusPending,
		AcceptanceCriteria: append([]string(nil), criteria...),
	}, nil
}

// NewStoryUpdate validates the full editable content of an existing story.
// A nil estimatedHours means "no estimate" and is valid.
func NewStoryUpdate(id, projectID, title, description, priority, status string, criteria []string, estimatedHours *float64) (Story, error) {
	fields := make(map[string]string)
	validateStoryContent(fields, title, description, priority, criteria)
	validateStatus(fields, status)
	validateEstimatedHours(fields, estimatedHours)
	if len(fields) > 0 {
		return Story{}, &ValidationError{Fields: fields}
	}

	var hours *float64
	if estimatedHours != nil {
		value := *estimatedHours
		hours = &value
	}
	return Story{
		ID:                 id,
		ProjectID:          projectID,
		Title:              title,
		Description:        description,
		Priority:           priority,
		Status:             status,
		AcceptanceCriteria: append([]string(nil), criteria...),
		EstimatedHours:     hours,
	}, nil
}

// validateStatus accumulates in fields the closed set of statuses.
func validateStatus(fields map[string]string, status string) {
	for _, allowed := range AllowedStatuses() {
		if status == allowed {
			return
		}
	}
	fields["status"] = "must be " + StatusPending + ", " + StatusInProgress + " or " + StatusCompleted
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

// validateStoryContent accumulates in fields the content rules shared by story
// creation and modification.
func validateStoryContent(fields map[string]string, title, description, priority string, criteria []string) {
	if strings.TrimSpace(title) == "" {
		fields["title"] = "is required"
	}
	if strings.TrimSpace(description) == "" {
		fields["description"] = "is required"
	}
	if priority != "alta" && priority != "media" && priority != "baja" {
		fields["priority"] = "must be alta, media or baja"
	}
	if len(criteria) == 0 {
		fields["acceptance_criteria"] = "must contain at least one non-blank criterion"
	} else {
		for _, criterion := range criteria {
			if strings.TrimSpace(criterion) == "" {
				fields["acceptance_criteria"] = "must contain only non-blank criteria"
				break
			}
		}
	}
}
