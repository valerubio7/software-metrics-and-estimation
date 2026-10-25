// Package domain contains the Defect entity and its validation rules.
package domain

import (
	"strings"
	"unicode/utf8"
)

const (
	// MaxDescriptionLength is the maximum description length in characters (runes), after trimming.
	MaxDescriptionLength = 500

	RequiredMessage          = "is required"
	NulMessage               = "must not contain NUL characters"
	DescriptionLengthMessage = "must be at most 500 characters"
	SeverityMessage          = "must be one of alta, media or baja"
	StatusMessage            = "must be one of detectado, en_correccion or resuelto"
	// InitialStatusMessage singles out resuelto: its transition needs the resolution
	// sprint registered by US-17, so creation only accepts the initial subcatalog.
	InitialStatusMessage = "must be detectado or en_correccion when registering a defect"
)

// Severities is the closed severity catalog; it mirrors the story priority values.
var Severities = []string{"alta", "media", "baja"}

// Statuses is the full storage catalog: the CHECK constraint of migration 000013 accepts
// resuelto, but creation applies only InitialStatuses (the transition belongs to US-17).
var Statuses = []string{"detectado", "en_correccion", "resuelto"}

// InitialStatuses is the subcatalog accepted by this registration use case.
var InitialStatuses = []string{"detectado", "en_correccion"}

// Defect is a problem detected during development, always scoped to one project with its
// related story and its detection sprint. Resolution data ships with US-17.
type Defect struct {
	ID                 string
	ProjectID          string
	StoryID            string
	SprintID           string // Sprint de detección
	Description        string // already trimmed
	Severity           string
	Status             string
	ResolutionSprintID string
}

// NewDefectParams groups the input already normalized by the application layer: all IDs
// arrive as canonical UUID strings and are never revalidated here.
type NewDefectParams struct {
	ID, ProjectID, StoryID, SprintID string
	Description                      string
	Severity                         string
	Status                           string
}

// ValidationError describes invalid defect input by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "defect validation failed" }

// NewDefect validates description (required, no NUL character, at most 500 characters after
// TrimSpace), severity (exact catalog value) and status (exact initial catalog value, which
// excludes resuelto because its resolution needs US-17). On failure it returns
// *ValidationError with keys "description", "severity" and/or "status"; all field errors
// arrive together. Description holds the trimmed value.
func NewDefect(params NewDefectParams) (Defect, error) {
	fields := make(map[string]string)

	description := strings.TrimSpace(params.Description)
	validateDescription(fields, description)
	validateSeverity(fields, params.Severity)
	validateStatus(fields, params.Status)

	if len(fields) > 0 {
		return Defect{}, &ValidationError{Fields: fields}
	}
	return Defect{
		ID:          params.ID,
		ProjectID:   params.ProjectID,
		StoryID:     params.StoryID,
		SprintID:    params.SprintID,
		Description: description,
		Severity:    params.Severity,
		Status:      params.Status,
	}, nil
}

// validateDescription expects an already trimmed description. NUL is rejected because
// PostgreSQL text cannot store 0x00 (SQLSTATE 22021) and it would surface as a 500.
func validateDescription(fields map[string]string, description string) {
	switch {
	case description == "":
		fields["description"] = RequiredMessage
	case strings.ContainsRune(description, 0):
		fields["description"] = NulMessage
	case utf8.RuneCountInString(description) > MaxDescriptionLength:
		fields["description"] = DescriptionLengthMessage
	}
}

// validateSeverity expects the catalog values verbatim: no trim and no case folding.
func validateSeverity(fields map[string]string, severity string) {
	if severity == "" {
		fields["severity"] = RequiredMessage
		return
	}
	if !containsExact(Severities, severity) {
		fields["severity"] = SeverityMessage
	}
}

// validateStatus expects the catalog values verbatim; resuelto is rejected by the
// registration subcatalog with a dedicated message that points at the US-17 transition.
func validateStatus(fields map[string]string, status string) {
	if status == "" {
		fields["status"] = RequiredMessage
		return
	}
	switch {
	case containsExact(Statuses, status) && !containsExact(InitialStatuses, status):
		fields["status"] = InitialStatusMessage
	case !containsExact(Statuses, status):
		fields["status"] = StatusMessage
	}
}

// containsExact reports whether value appears verbatim in the catalog slice.
func containsExact(catalog []string, value string) bool {
	for _, entry := range catalog {
		if entry == value {
			return true
		}
	}
	return false
}
