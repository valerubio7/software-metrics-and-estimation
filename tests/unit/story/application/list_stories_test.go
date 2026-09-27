package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

const listProjectID = "0f8fad5b-d9cb-469f-a165-70867728950e"

// storyLister is a hand-written fake of the read-only StoryLister port. It counts
// calls, captures the project identifier it receives and returns a preconfigured
// result. It exposes no write method, so a use case cannot modify through it.
type storyLister struct {
	calls    int
	received string
	stories  []domain.Story
	err      error
}

func (l *storyLister) ListByProject(_ context.Context, projectID string) ([]domain.Story, error) {
	l.calls++
	l.received = projectID
	return l.stories, l.err
}

func listedStory(id, priority string) domain.Story {
	return domain.Story{ID: id, ProjectID: listProjectID, Priority: priority}
}

func listedIDs(stories []domain.Story) []string {
	ids := make([]string, 0, len(stories))
	for _, story := range stories {
		ids = append(ids, story.ID)
	}
	return ids
}

func TestListStoriesRejectsInvalidProjectIDWithoutTouchingThePort(t *testing.T) {
	lister := &storyLister{}

	backlog, err := application.NewListStoriesUseCase(lister).Execute(
		context.Background(), application.ListStoriesQuery{ProjectID: "no-es-uuid"})

	assertValidationFields(t, err, map[string]string{"project_id": "must be a valid UUID"})
	if lister.calls != 0 {
		t.Errorf("lister calls = %d, want none", lister.calls)
	}
	if !reflect.DeepEqual(backlog, domain.Backlog{}) {
		t.Errorf("backlog = %#v, want the zero Backlog", backlog)
	}
}

func TestListStoriesReadsOnceAndReturnsTheOrderedBacklog(t *testing.T) {
	// The port returns creation order; the use case must order by priority.
	lister := &storyLister{stories: []domain.Story{
		listedStory("S1", "media"), listedStory("S2", "alta"), listedStory("S3", "media"),
		listedStory("S4", "baja"), listedStory("S5", "alta"),
	}}

	backlog, err := application.NewListStoriesUseCase(lister).Execute(
		context.Background(), application.ListStoriesQuery{ProjectID: listProjectID})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if lister.calls != 1 {
		t.Fatalf("lister calls = %d, want exactly one", lister.calls)
	}
	if backlog.ProjectID != listProjectID {
		t.Errorf("ProjectID = %q, want %q", backlog.ProjectID, listProjectID)
	}
	if got, want := listedIDs(backlog.Stories), []string{"S2", "S5", "S1", "S3", "S4"}; !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestListStoriesRejectsHostileProjectIdentifiers(t *testing.T) {
	for _, projectID := range []string{
		"no-es-uuid", "abc", "123", "0f8fad5b-d9cb-469f-a165-7086772895", "",
	} {
		t.Run("id="+projectID, func(t *testing.T) {
			lister := &storyLister{stories: []domain.Story{listedStory("S1", "alta")}}

			backlog, err := application.NewListStoriesUseCase(lister).Execute(
				context.Background(), application.ListStoriesQuery{ProjectID: projectID})

			assertValidationFields(t, err, map[string]string{"project_id": "must be a valid UUID"})
			if lister.calls != 0 {
				t.Errorf("lister calls = %d, want none", lister.calls)
			}
			if len(backlog.Stories) != 0 {
				t.Errorf("stories = %v, want none", listedIDs(backlog.Stories))
			}
		})
	}
}

func TestListStoriesCanonicalizesTheProjectIdentifier(t *testing.T) {
	lister := &storyLister{stories: []domain.Story{listedStory("S1", "alta")}}

	backlog, err := application.NewListStoriesUseCase(lister).Execute(
		context.Background(), application.ListStoriesQuery{ProjectID: "0F8FAD5B-D9CB-469F-A165-70867728950E"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if lister.calls != 1 || lister.received != listProjectID {
		t.Errorf("port received %d call(s) with %q, want one call with %q", lister.calls, lister.received, listProjectID)
	}
	if backlog.ProjectID != listProjectID {
		t.Errorf("ProjectID = %q, want the canonical %q", backlog.ProjectID, listProjectID)
	}
}

func TestListStoriesPropagatesPortErrorsWithoutPartialBacklog(t *testing.T) {
	unexpected := errors.New("connection reset")
	tests := []struct {
		name string
		err  error
	}{
		{name: "project not found", err: application.ErrProjectNotFound},
		{name: "unexpected error", err: unexpected},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// The fake returns stories together with the error: none may leak out.
			lister := &storyLister{stories: []domain.Story{listedStory("S1", "alta")}, err: test.err}

			backlog, err := application.NewListStoriesUseCase(lister).Execute(
				context.Background(), application.ListStoriesQuery{ProjectID: listProjectID})

			if !errors.Is(err, test.err) {
				t.Errorf("error = %v, want %v", err, test.err)
			}
			if errors.Is(err, application.ErrProjectNotFound) != (test.err == application.ErrProjectNotFound) {
				t.Errorf("error %v was confused with ErrProjectNotFound", err)
			}
			if !reflect.DeepEqual(backlog, domain.Backlog{}) {
				t.Errorf("backlog = %#v, want the zero Backlog (no partial list)", backlog)
			}
		})
	}
}

func TestListStoriesProjectWithoutStoriesIsNotAnError(t *testing.T) {
	lister := &storyLister{stories: nil}

	backlog, err := application.NewListStoriesUseCase(lister).Execute(
		context.Background(), application.ListStoriesQuery{ProjectID: listProjectID})

	if err != nil {
		t.Fatalf("Execute() error = %v, want none for a project without stories", err)
	}
	if lister.calls != 1 {
		t.Errorf("lister calls = %d, want exactly one", lister.calls)
	}
	if backlog.ProjectID != listProjectID || len(backlog.Stories) != 0 {
		t.Errorf("backlog = %#v, want the project with no stories", backlog)
	}
}
