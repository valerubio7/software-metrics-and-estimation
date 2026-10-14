package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

const listProjectID = "82d38423-f02d-4259-9e35-4a291585bb1e"

// historyReader is a read-only fake of the SprintHistoryReader port.
type historyReader struct {
	calls     int
	projectID string
	sprints   []domain.ClosedSprint
	err       error
}

func (r *historyReader) ListClosedByProject(_ context.Context, projectID string) ([]domain.ClosedSprint, error) {
	r.calls++
	r.projectID = projectID
	return r.sprints, r.err
}

func TestListSprintsRejectsInvalidProjectIDWithoutReadingStorage(t *testing.T) {
	reader := &historyReader{}
	useCase := application.NewListSprintsUseCase(reader)

	history, err := useCase.Execute(context.Background(), application.ListSprintsQuery{ProjectID: "no-es-uuid"})

	var validation *domain.ValidationError
	if !errors.As(err, &validation) || validation.Fields["project_id"] != "must be a valid UUID" {
		t.Fatalf("Execute() error = %v, want ValidationError with fields.project_id", err)
	}
	if reader.calls != 0 {
		t.Errorf("reader calls = %d, want 0", reader.calls)
	}
	if !reflect.DeepEqual(history, domain.SprintHistory{}) {
		t.Errorf("history = %#v, want the zero value", history)
	}
}

func TestListSprintsReturnsTheHistoryOfAValidProject(t *testing.T) {
	completedAt := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	points := 5
	sprints := []domain.ClosedSprint{{
		ID: "sprint-1", ProjectID: listProjectID, SprintGoal: "Deliver the login",
		Stories: []domain.PlannedStory{{ID: "story-1", Title: "Login", Priority: "alta", StoryPoints: &points, CompletedAt: &completedAt}},
	}}
	reader := &historyReader{sprints: sprints}
	useCase := application.NewListSprintsUseCase(reader)

	history, err := useCase.Execute(context.Background(), application.ListSprintsQuery{ProjectID: listProjectID})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if reader.calls != 1 || reader.projectID != listProjectID {
		t.Errorf("reader calls/projectID = %d/%q, want 1/%q", reader.calls, reader.projectID, listProjectID)
	}
	want := domain.SprintHistory{ProjectID: listProjectID, Sprints: sprints}
	if !reflect.DeepEqual(history, want) {
		t.Errorf("history = %#v, want %#v", history, want)
	}
}

func TestListSprintsRejectsEveryHostileProjectIDWithoutReadingStorage(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
	}{
		{name: "plain text", projectID: "no-es-uuid"},
		{name: "short text", projectID: "abc"},
		{name: "digits only", projectID: "123"},
		{name: "truncated UUID", projectID: "82d38423-f02d-4259-9e35-4a291585bb1"},
		{name: "empty", projectID: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &historyReader{}
			useCase := application.NewListSprintsUseCase(reader)

			history, err := useCase.Execute(context.Background(), application.ListSprintsQuery{ProjectID: tt.projectID})

			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Fields["project_id"] == "" {
				t.Fatalf("Execute(%q) error = %v, want ValidationError with fields.project_id", tt.projectID, err)
			}
			if reader.calls != 0 {
				t.Errorf("reader calls = %d, want 0", reader.calls)
			}
			if !reflect.DeepEqual(history, domain.SprintHistory{}) {
				t.Errorf("history = %#v, want the zero value", history)
			}
		})
	}
}

func TestListSprintsPassesTheCanonicalIdentifierToThePortAndResult(t *testing.T) {
	reader := &historyReader{}
	useCase := application.NewListSprintsUseCase(reader)

	history, err := useCase.Execute(context.Background(), application.ListSprintsQuery{ProjectID: "82D38423-F02D-4259-9E35-4A291585BB1E"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if reader.calls != 1 || reader.projectID != listProjectID {
		t.Errorf("reader calls/projectID = %d/%q, want 1/%q", reader.calls, reader.projectID, listProjectID)
	}
	if history.ProjectID != listProjectID {
		t.Errorf("history.ProjectID = %q, want canonical %q", history.ProjectID, listProjectID)
	}
}

func TestListSprintsPreservesTheReaderOrder(t *testing.T) {
	sprints := []domain.ClosedSprint{
		{ID: "newest", ProjectID: listProjectID, SprintGoal: "C", Stories: []domain.PlannedStory{{ID: "s1"}, {ID: "s2"}, {ID: "s3"}}},
		{ID: "middle", ProjectID: listProjectID, SprintGoal: "B"},
		{ID: "oldest", ProjectID: listProjectID, SprintGoal: "A"},
	}
	reader := &historyReader{sprints: sprints}
	useCase := application.NewListSprintsUseCase(reader)

	history, err := useCase.Execute(context.Background(), application.ListSprintsQuery{ProjectID: listProjectID})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	var sprintIDs, storyIDs []string
	for _, sprint := range history.Sprints {
		sprintIDs = append(sprintIDs, sprint.ID)
	}
	for _, story := range history.Sprints[0].Stories {
		storyIDs = append(storyIDs, story.ID)
	}
	if !reflect.DeepEqual(sprintIDs, []string{"newest", "middle", "oldest"}) || !reflect.DeepEqual(storyIDs, []string{"s1", "s2", "s3"}) {
		t.Errorf("sprint/story order = %v/%v, want reader order unchanged", sprintIDs, storyIDs)
	}
}

func TestListSprintsPropagatesPortErrorsWithoutAPartialList(t *testing.T) {
	leftover := []domain.ClosedSprint{{ID: "partial", ProjectID: listProjectID, SprintGoal: "Must not leak"}}
	unexpected := errors.New("connection reset")
	tests := []struct {
		name string
		err  error
	}{
		{name: "project not found", err: application.ErrProjectNotFound},
		{name: "unexpected failure", err: unexpected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &historyReader{sprints: leftover, err: tt.err}
			useCase := application.NewListSprintsUseCase(reader)

			history, err := useCase.Execute(context.Background(), application.ListSprintsQuery{ProjectID: listProjectID})

			if !errors.Is(err, tt.err) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.err)
			}
			if reader.calls != 1 {
				t.Errorf("reader calls = %d, want 1", reader.calls)
			}
			if !reflect.DeepEqual(history, domain.SprintHistory{}) {
				t.Errorf("history = %#v, want the zero value, never a partial list", history)
			}
		})
	}
}
