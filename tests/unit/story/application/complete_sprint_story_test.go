package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
)

type fakeSprintStoryCompleter struct {
	calls                        int
	projectID, sprintID, storyID string
	result                       application.SprintStoryCompletion
	err                          error
}

func (f *fakeSprintStoryCompleter) CompleteSprintStory(_ context.Context, projectID, sprintID, storyID string) (application.SprintStoryCompletion, error) {
	f.calls++
	f.projectID, f.sprintID, f.storyID = projectID, sprintID, storyID
	return f.result, f.err
}

func TestCompleteSprintStoryUseCaseExecute(t *testing.T) {
	command := application.CompleteSprintStoryCommand{ProjectID: "project-1", SprintID: "sprint-1", StoryID: "story-1"}

	t.Run("delegates once with the command IDs and returns the completion unchanged", func(t *testing.T) {
		want := application.SprintStoryCompletion{
			ProjectID: "project-1", SprintID: "sprint-1", StoryID: "story-1",
			CompletedAt: time.Date(2026, 10, 4, 14, 3, 21, 123456000, time.UTC),
		}
		completer := &fakeSprintStoryCompleter{result: want}

		got, err := application.NewCompleteSprintStoryUseCase(completer).Execute(context.Background(), command)

		if err != nil || got != want {
			t.Fatalf("Execute() = %+v, %v; want %+v, nil", got, err, want)
		}
		if completer.calls != 1 || completer.projectID != "project-1" || completer.sprintID != "sprint-1" || completer.storyID != "story-1" {
			t.Fatalf("port call = %d (%q, %q, %q), want one call with the command IDs", completer.calls, completer.projectID, completer.sprintID, completer.storyID)
		}
	})

	errorCases := []struct {
		name string
		err  error
	}{
		{"project not found", application.ErrProjectNotFound},
		{"sprint not found", application.ErrSprintNotFound},
		{"story not found", application.ErrStoryNotFound},
		{"sprint closed", application.ErrSprintClosed},
		{"story not in sprint", application.ErrStoryNotInSprint},
		{"story already completed", application.ErrStoryAlreadyCompleted},
		{"unexpected", errors.New("database unavailable")},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			completer := &fakeSprintStoryCompleter{err: tc.err}

			got, err := application.NewCompleteSprintStoryUseCase(completer).Execute(context.Background(), command)

			// Identity (not only errors.Is) proves the use case does not wrap the port error.
			if err != tc.err {
				t.Fatalf("Execute() error = %v, want %v unwrapped", err, tc.err)
			}
			if got != (application.SprintStoryCompletion{}) {
				t.Fatalf("Execute() result = %+v, want zero value", got)
			}
			if completer.calls != 1 {
				t.Fatalf("port calls = %d, want 1", completer.calls)
			}
		})
	}
}
