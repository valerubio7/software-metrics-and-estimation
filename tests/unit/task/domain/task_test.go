package domain_test

import (
	"errors"
	"math"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

const estimatedHoursMessage = "must be greater than 0, at most 99999.99 and have at most 2 decimals"

func TestNewTask(t *testing.T) {
	t.Run("valid task preserves IDs and untrimmed title with nil estimate", func(t *testing.T) {
		task, err := domain.NewTask("task-id", "project-id", "sprint-id", "story-id", "  Diseñar  ", nil)
		if err != nil {
			t.Fatalf("NewTask() error = %v", err)
		}
		if task.ID != "task-id" || task.ProjectID != "project-id" || task.SprintID != "sprint-id" ||
			task.StoryID != "story-id" || task.Title != "  Diseñar  " {
			t.Errorf("task fields = %#v, want unchanged inputs", task)
		}
		if task.EstimatedHours != nil {
			t.Errorf("EstimatedHours = %v, want nil", *task.EstimatedHours)
		}
	})

	t.Run("valid estimates are accepted and copied, not aliased", func(t *testing.T) {
		for _, hours := range []float64{0.01, 1.5, 4.25, 99999.99} {
			original := hours
			task, err := domain.NewTask("task-id", "project-id", "sprint-id", "story-id", "Título", &original)
			if err != nil {
				t.Fatalf("NewTask(%v) error = %v", hours, err)
			}
			if task.EstimatedHours == nil || *task.EstimatedHours != hours {
				t.Fatalf("EstimatedHours = %v, want %v", task.EstimatedHours, hours)
			}
			if task.EstimatedHours == &original {
				t.Fatalf("EstimatedHours aliases the caller's pointer")
			}
			original = -1
			if *task.EstimatedHours != hours {
				t.Errorf("mutating the caller's pointer changed the task: %v, want %v", *task.EstimatedHours, hours)
			}
		}
	})

	t.Run("invalid title", func(t *testing.T) {
		for _, title := range []string{"", "   ", "\t\n"} {
			_, err := domain.NewTask("task-id", "project-id", "sprint-id", "story-id", title, nil)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("NewTask(title=%q) error = %v, want ValidationError", title, err)
			}
			if validation.Fields["title"] != "is required" {
				t.Errorf("Fields[title] = %q, want %q", validation.Fields["title"], "is required")
			}
		}
	})

	t.Run("invalid estimate", func(t *testing.T) {
		for _, hours := range []float64{0, -1, 100000, 99999.991, 1.234, math.NaN(), math.Inf(1)} {
			hours := hours
			_, err := domain.NewTask("task-id", "project-id", "sprint-id", "story-id", "Título", &hours)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("NewTask(hours=%v) error = %v, want ValidationError", hours, err)
			}
			if validation.Fields["estimated_hours"] != estimatedHoursMessage {
				t.Errorf("Fields[estimated_hours] = %q, want %q", validation.Fields["estimated_hours"], estimatedHoursMessage)
			}
		}
	})

	t.Run("both fields invalid at once accumulate both keys", func(t *testing.T) {
		invalidHours := -1.0
		_, err := domain.NewTask("task-id", "project-id", "sprint-id", "story-id", "   ", &invalidHours)
		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("NewTask() error = %v, want ValidationError", err)
		}
		if len(validation.Fields) != 2 || validation.Fields["title"] != "is required" || validation.Fields["estimated_hours"] != estimatedHoursMessage {
			t.Errorf("Fields = %#v, want title and estimated_hours", validation.Fields)
		}
		if validation.Error() != "task validation failed" {
			t.Errorf("Error() = %q, want %q", validation.Error(), "task validation failed")
		}
	})
}
