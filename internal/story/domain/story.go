package domain

import "strings"

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
	if len(fields) > 0 {
		return Story{}, &ValidationError{Fields: fields}
	}

	return Story{
		ID:                 id,
		ProjectID:          projectID,
		Title:              title,
		Description:        description,
		Priority:           priority,
		Status:             "pendiente",
		AcceptanceCriteria: append([]string(nil), criteria...),
	}, nil
}
