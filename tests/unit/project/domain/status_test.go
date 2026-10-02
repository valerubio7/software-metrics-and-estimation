package domain_test

import (
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

func TestDeriveStatusUsesInclusiveCalendarDates(t *testing.T) {
	start := time.Date(2026, time.March, 1, 22, 0, 0, 0, time.FixedZone("west", -8*60*60))
	finish := time.Date(2026, time.March, 3, 1, 0, 0, 0, time.UTC)
	project := domain.Project{StartDate: start, PlannedFinishDate: finish}
	tests := []struct {
		name string
		date time.Time
		want domain.ProjectStatus
	}{
		{"day before", time.Date(2026, time.February, 28, 23, 0, 0, 0, time.UTC), domain.ProjectStatusPlanned},
		{"start inclusive", time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC), domain.ProjectStatusActive},
		{"finish inclusive", time.Date(2026, time.March, 3, 23, 0, 0, 0, time.UTC), domain.ProjectStatusActive},
		{"day after", time.Date(2026, time.March, 4, 0, 0, 0, 0, time.UTC), domain.ProjectStatusOverdue},
		{"different year", time.Date(2025, time.March, 2, 0, 0, 0, 0, time.UTC), domain.ProjectStatusPlanned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := project.DeriveStatus(tt.date); got != tt.want {
				t.Errorf("DeriveStatus(%v) = %q, want %q", tt.date, got, tt.want)
			}
		})
	}
}

func TestDeriveStatusSingleDayProjectIsActiveOnThatDate(t *testing.T) {
	day := time.Date(2026, time.July, 9, 0, 0, 0, 0, time.UTC)
	project := domain.Project{StartDate: day, PlannedFinishDate: day}
	if got := project.DeriveStatus(day.Add(23 * time.Hour)); got != domain.ProjectStatusActive {
		t.Fatalf("DeriveStatus() = %q, want %q", got, domain.ProjectStatusActive)
	}
}
