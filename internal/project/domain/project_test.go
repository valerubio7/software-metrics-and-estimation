package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-estimation/internal/project/domain"
)

func TestNewProjectAcceptsEqualDates(t *testing.T) {
	date := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	project, err := domain.NewProject("project-id", "Metrics portal", date, date)

	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if project.ID != "project-id" {
		t.Errorf("ID = %q, want %q", project.ID, "project-id")
	}
}

func TestNewProjectRejectsInvalidBasicData(t *testing.T) {
	date := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		projectName string
		startDate  time.Time
		finishDate time.Time
		field      string
	}{
		{name: "blank name", projectName: " \t", startDate: date, finishDate: date, field: "name"},
		{name: "missing start date", projectName: "Metrics portal", finishDate: date, field: "start_date"},
		{name: "missing finish date", projectName: "Metrics portal", startDate: date, field: "planned_finish_date"},
		{name: "finish before start", projectName: "Metrics portal", startDate: date, finishDate: date.AddDate(0, 0, -1), field: "planned_finish_date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewProject("project-id", tt.projectName, tt.startDate, tt.finishDate)

			var validationErr *domain.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("NewProject() error = %v, want ValidationError", err)
			}
			if _, ok := validationErr.Fields[tt.field]; !ok {
				t.Errorf("validation fields = %#v, want %q", validationErr.Fields, tt.field)
			}
		})
	}
}
