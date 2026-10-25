package domain_test

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/domain"
)

const (
	hoursMessage       = "must be greater than 0, at most 24 and have at most 2 decimals"
	futureDateMessage  = "must not be in the future"
	descriptionMessage = "must be at most 500 characters"
)

var today = time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)

func floatPtr(v float64) *float64 { return &v }

func validParams() domain.NewEffortRecordParams {
	return domain.NewEffortRecordParams{
		ID:          "record-id",
		ProjectID:   "project-id",
		StoryID:     "story-id",
		MemberID:    "member-id",
		WorkDate:    today,
		Description: "Revisión de código",
		Hours:       floatPtr(2.5),
	}
}

func TestNewEffortRecord(t *testing.T) {
	t.Run("valid record over a story preserves IDs, work date and nil task", func(t *testing.T) {
		record, err := domain.NewEffortRecord(validParams(), today)
		if err != nil {
			t.Fatalf("NewEffortRecord() error = %v", err)
		}
		if record.ID != "record-id" || record.ProjectID != "project-id" || record.StoryID != "story-id" ||
			record.MemberID != "member-id" {
			t.Errorf("record IDs = %#v, want unchanged inputs", record)
		}
		if record.TaskID != nil {
			t.Errorf("TaskID = %q, want nil", *record.TaskID)
		}
		if !record.WorkDate.Equal(today) {
			t.Errorf("WorkDate = %v, want %v", record.WorkDate, today)
		}
		if record.Description != "Revisión de código" || record.Hours != 2.5 {
			t.Errorf("Description/Hours = %q/%v, want %q/%v", record.Description, record.Hours, "Revisión de código", 2.5)
		}
	})

	t.Run("valid record over a task preserves the task ID", func(t *testing.T) {
		params := validParams()
		taskID := "task-id"
		params.TaskID = &taskID
		record, err := domain.NewEffortRecord(params, today)
		if err != nil {
			t.Fatalf("NewEffortRecord() error = %v", err)
		}
		if record.TaskID == nil || *record.TaskID != "task-id" {
			t.Errorf("TaskID = %v, want task-id", record.TaskID)
		}
	})

	t.Run("description is required", func(t *testing.T) {
		for _, description := range []string{"", "   ", "\t\n"} {
			params := validParams()
			params.Description = description
			_, err := domain.NewEffortRecord(params, today)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("NewEffortRecord(description=%q) error = %v, want ValidationError", description, err)
			}
			if validation.Fields["description"] != "is required" {
				t.Errorf("Fields[description] = %q, want %q", validation.Fields["description"], "is required")
			}
		}
	})

	t.Run("description is stored trimmed", func(t *testing.T) {
		params := validParams()
		params.Description = "  Revisión de código  "
		record, err := domain.NewEffortRecord(params, today)
		if err != nil {
			t.Fatalf("NewEffortRecord() error = %v", err)
		}
		if record.Description != "Revisión de código" {
			t.Errorf("Description = %q, want %q", record.Description, "Revisión de código")
		}
	})

	t.Run("hours are required", func(t *testing.T) {
		params := validParams()
		params.Hours = nil
		_, err := domain.NewEffortRecord(params, today)
		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("NewEffortRecord(hours=nil) error = %v, want ValidationError", err)
		}
		if validation.Fields["hours"] != "is required" {
			t.Errorf("Fields[hours] = %q, want %q", validation.Fields["hours"], "is required")
		}
	})

	t.Run("invalid hours", func(t *testing.T) {
		for _, hours := range []float64{0, -1, -0.01, 24.01, 100, 1.005, 0.001, math.NaN(), math.Inf(1)} {
			params := validParams()
			params.Hours = floatPtr(hours)
			_, err := domain.NewEffortRecord(params, today)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("NewEffortRecord(hours=%v) error = %v, want ValidationError", hours, err)
			}
			if validation.Fields["hours"] != hoursMessage {
				t.Errorf("Fields[hours] = %q, want %q", validation.Fields["hours"], hoursMessage)
			}
		}
	})

	t.Run("work date after today is rejected", func(t *testing.T) {
		params := validParams()
		params.WorkDate = today.AddDate(0, 0, 1)
		_, err := domain.NewEffortRecord(params, today)
		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("NewEffortRecord(future) error = %v, want ValidationError", err)
		}
		if validation.Fields["work_date"] != futureDateMessage {
			t.Errorf("Fields[work_date] = %q, want %q", validation.Fields["work_date"], futureDateMessage)
		}
	})

	t.Run("description of exactly 500 multibyte characters is accepted", func(t *testing.T) {
		params := validParams()
		params.Description = strings.Repeat("ñ", domain.MaxDescriptionLength) // 1000 bytes, 500 runes
		record, err := domain.NewEffortRecord(params, today)
		if err != nil {
			t.Fatalf("NewEffortRecord(500 runes) error = %v", err)
		}
		if record.Description != params.Description {
			t.Errorf("Description changed, want the 500-rune text preserved")
		}
	})

	t.Run("description of 501 characters is rejected counting runes, not bytes", func(t *testing.T) {
		for name, description := range map[string]string{
			"multibyte":           strings.Repeat("ñ", domain.MaxDescriptionLength+1),
			"ascii":               strings.Repeat("a", domain.MaxDescriptionLength+1),
			"padded then trimmed": "  " + strings.Repeat("ñ", domain.MaxDescriptionLength+1) + "  ",
		} {
			params := validParams()
			params.Description = description
			_, err := domain.NewEffortRecord(params, today)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("%s: NewEffortRecord(501 runes) error = %v, want ValidationError", name, err)
			}
			if validation.Fields["description"] != descriptionMessage {
				t.Errorf("%s: Fields[description] = %q, want %q", name, validation.Fields["description"], descriptionMessage)
			}
		}
	})

	t.Run("padding does not count towards the description limit", func(t *testing.T) {
		params := validParams()
		params.Description = "   " + strings.Repeat("a", domain.MaxDescriptionLength) + "   "
		record, err := domain.NewEffortRecord(params, today)
		if err != nil {
			t.Fatalf("NewEffortRecord(500 runes + padding) error = %v", err)
		}
		if utf8.RuneCountInString(record.Description) != domain.MaxDescriptionLength {
			t.Errorf("Description has %d runes, want %d", utf8.RuneCountInString(record.Description), domain.MaxDescriptionLength)
		}
	})

	t.Run("valid hours are accepted and stored as given", func(t *testing.T) {
		for _, hours := range []float64{0.01, 0.5, 7.25, 8, 24} {
			params := validParams()
			params.Hours = floatPtr(hours)
			record, err := domain.NewEffortRecord(params, today)
			if err != nil {
				t.Fatalf("NewEffortRecord(hours=%v) error = %v", hours, err)
			}
			if record.Hours != hours {
				t.Errorf("Hours = %v, want %v", record.Hours, hours)
			}
		}
	})

	t.Run("hours with more than two decimals are rejected even above the minimum", func(t *testing.T) {
		params := validParams()
		params.Hours = floatPtr(2.555)
		_, err := domain.NewEffortRecord(params, today)
		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("NewEffortRecord(hours=2.555) error = %v, want ValidationError", err)
		}
		if validation.Fields["hours"] != hoursMessage {
			t.Errorf("Fields[hours] = %q, want %q", validation.Fields["hours"], hoursMessage)
		}
	})

	t.Run("several invalid fields are reported together", func(t *testing.T) {
		params := validParams()
		params.Description = "   "
		params.Hours = floatPtr(0)
		params.WorkDate = today.AddDate(0, 0, 1)
		_, err := domain.NewEffortRecord(params, today)
		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("NewEffortRecord() error = %v, want *ValidationError", err)
		}
		want := map[string]string{
			"description": "is required",
			"hours":       hoursMessage,
			"work_date":   futureDateMessage,
		}
		if len(validation.Fields) != len(want) {
			t.Fatalf("Fields = %v, want exactly %v", validation.Fields, want)
		}
		for key, message := range want {
			if validation.Fields[key] != message {
				t.Errorf("Fields[%s] = %q, want %q", key, validation.Fields[key], message)
			}
		}
		if validation.Error() != "effort record validation failed" {
			t.Errorf("Error() = %q, want %q", validation.Error(), "effort record validation failed")
		}
	})

	t.Run("zero work date is not revalidated and produces no work_date error", func(t *testing.T) {
		params := validParams()
		params.WorkDate = time.Time{}
		record, err := domain.NewEffortRecord(params, today)
		if err != nil {
			t.Fatalf("NewEffortRecord(zero work date) error = %v", err)
		}
		if !record.WorkDate.IsZero() {
			t.Errorf("WorkDate = %v, want zero", record.WorkDate)
		}

		params.Description = ""
		_, err = domain.NewEffortRecord(params, today)
		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("NewEffortRecord() error = %v, want *ValidationError", err)
		}
		if _, found := validation.Fields["work_date"]; found {
			t.Errorf("Fields = %v, want no work_date key for a zero date", validation.Fields)
		}
	})

	t.Run("task ID is copied without aliasing the caller's pointer", func(t *testing.T) {
		params := validParams()
		original := "task-id"
		params.TaskID = &original
		record, err := domain.NewEffortRecord(params, today)
		if err != nil {
			t.Fatalf("NewEffortRecord() error = %v", err)
		}
		if record.TaskID == &original {
			t.Fatalf("TaskID aliases the caller's pointer")
		}
		original = "mutated"
		if record.TaskID == nil || *record.TaskID != "task-id" {
			t.Errorf("TaskID = %v after mutating the original, want task-id", record.TaskID)
		}
	})

	t.Run("work date equal to or before today is accepted", func(t *testing.T) {
		for _, workDate := range []time.Time{today, today.AddDate(0, 0, -1), today.AddDate(-1, 0, 0)} {
			params := validParams()
			params.WorkDate = workDate
			record, err := domain.NewEffortRecord(params, today)
			if err != nil {
				t.Fatalf("NewEffortRecord(workDate=%v) error = %v", workDate, err)
			}
			if !record.WorkDate.Equal(workDate) {
				t.Errorf("WorkDate = %v, want %v", record.WorkDate, workDate)
			}
		}
	})
}
