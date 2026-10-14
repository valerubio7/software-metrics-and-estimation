package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

func TestNewSprintPreservesValidGoalsWithoutAdditionalContentRules(t *testing.T) {
	tests := []struct {
		name string
		goal string
	}{
		{name: "surrounding whitespace", goal: "  Deliver the initial metrics flow  "},
		{name: "arbitrary long content", goal: "GOAL\nwith punctuation !? and a deliberately long value " +
			"that must not be rejected or rewritten by undocumented content rules"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sprint, err := domain.NewSprint("sprint-id", "project-id", tt.goal)
			if err != nil {
				t.Fatalf("NewSprint() error = %v", err)
			}
			if sprint.ID != "sprint-id" || sprint.ProjectID != "project-id" || sprint.SprintGoal != tt.goal {
				t.Errorf("sprint = %#v, want IDs and goal preserved exactly", sprint)
			}
		})
	}
}

func TestNewSprintRejectsMissingOrBlankGoal(t *testing.T) {
	tests := []struct {
		name string
		goal string
	}{
		{name: "missing goal"},
		{name: "whitespace-only goal", goal: " \t\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewSprint("sprint-id", "project-id", tt.goal)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("NewSprint() error = %v, want ValidationError", err)
			}
			if validation.Fields["sprint_goal"] != "is required" {
				t.Errorf("validation fields = %#v, want sprint_goal required", validation.Fields)
			}
		})
	}
}

func TestPlannedStoryCompletedFollowsTheCompletionMark(t *testing.T) {
	mark := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	zeroMark := time.Time{}
	tests := []struct {
		name        string
		completedAt *time.Time
		want        bool
	}{
		{name: "no completion in the sprint", completedAt: nil, want: false},
		{name: "completion mark present", completedAt: &mark, want: true},
		{name: "zero time is still a recorded mark", completedAt: &zeroMark, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story := domain.PlannedStory{ID: "story-id", Title: "Story", Priority: "alta", CompletedAt: tt.completedAt}
			if got := story.Completed(); got != tt.want {
				t.Errorf("Completed() = %v, want %v", got, tt.want)
			}
		})
	}
}
