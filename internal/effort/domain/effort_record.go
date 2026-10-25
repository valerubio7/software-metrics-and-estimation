// Package domain contains the EffortRecord entity and its validation rules.
package domain

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MaxHours is the largest amount of hours a single effort record may hold.
	MaxHours = 24.0
	// MaxDescriptionLength is the maximum description length in characters (runes), after trimming.
	MaxDescriptionLength = 500

	maxHoursDecimals = 2

	requiredMessage    = "is required"
	hoursMessage       = "must be greater than 0, at most 24 and have at most 2 decimals"
	descriptionMessage = "must be at most 500 characters"
	nulMessage         = "must not contain NUL characters"
	futureDateMessage  = "must not be in the future"
)

// EffortRecord is actual effort registered by a project member. It always belongs to
// StoryID; TaskID only narrows the target within that story (nil means the story itself).
type EffortRecord struct {
	ID          string
	ProjectID   string
	StoryID     string
	TaskID      *string
	MemberID    string
	WorkDate    time.Time // civil date at UTC midnight
	Description string    // already trimmed
	Hours       float64
}

// NewEffortRecordParams groups the input already normalized by the application layer.
type NewEffortRecordParams struct {
	ID, ProjectID, StoryID, MemberID string
	TaskID                           *string
	WorkDate                         time.Time // zero means the application already reported it; it is not revalidated
	Description                      string
	Hours                            *float64 // nil means absent
}

// ValidationError describes invalid effort record input by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "effort record validation failed" }

// NewEffortRecord validates description (required, no NUL character, at most 500 characters after TrimSpace),
// hours (required, > 0, <= 24, at most 2 decimals) and work date (not after today, which is
// a UTC-midnight civil date). On failure it returns *ValidationError with keys "description",
// "hours" and/or "work_date". The returned record holds the trimmed description and a copy
// of TaskID, never the caller's pointer.
func NewEffortRecord(params NewEffortRecordParams, today time.Time) (EffortRecord, error) {
	fields := make(map[string]string)

	description := strings.TrimSpace(params.Description)
	validateDescription(fields, description)
	validateHours(fields, params.Hours)
	validateWorkDate(fields, params.WorkDate, today)

	if len(fields) > 0 {
		return EffortRecord{}, &ValidationError{Fields: fields}
	}

	var taskID *string
	if params.TaskID != nil {
		value := *params.TaskID
		taskID = &value
	}
	return EffortRecord{
		ID:          params.ID,
		ProjectID:   params.ProjectID,
		StoryID:     params.StoryID,
		TaskID:      taskID,
		MemberID:    params.MemberID,
		WorkDate:    params.WorkDate,
		Description: description,
		Hours:       *params.Hours,
	}, nil
}

// validateDescription expects an already trimmed description. NUL is rejected because
// PostgreSQL text cannot store 0x00 (SQLSTATE 22021) and it would surface as a 500.
func validateDescription(fields map[string]string, description string) {
	switch {
	case description == "":
		fields["description"] = requiredMessage
	case strings.ContainsRune(description, 0):
		fields["description"] = nulMessage
	case utf8.RuneCountInString(description) > MaxDescriptionLength:
		fields["description"] = descriptionMessage
	}
}

func validateHours(fields map[string]string, hours *float64) {
	if hours == nil {
		fields["hours"] = requiredMessage
		return
	}
	value := *hours
	if !(value > 0) || value > MaxHours || decimalPlaces(value) > maxHoursDecimals {
		fields["hours"] = hoursMessage
	}
}

// validateWorkDate skips the zero date: the application layer already reported it.
func validateWorkDate(fields map[string]string, workDate, today time.Time) {
	if !workDate.IsZero() && workDate.After(today) {
		fields["work_date"] = futureDateMessage
	}
}

// decimalPlaces counts decimals on the shortest decimal representation that round-trips the
// float64, not with float arithmetic. It mirrors the technique used by the task domain.
func decimalPlaces(value float64) int {
	_, fraction, found := strings.Cut(strconv.FormatFloat(value, 'f', -1, 64), ".")
	if !found {
		return 0
	}
	return len(fraction)
}
