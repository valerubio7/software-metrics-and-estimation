package application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
)

type assignmentRepository struct {
	calls     int
	sprintID  string
	projectID string
	storyIDs  []string
	err       error
}

func (r *assignmentRepository) AssignStories(_ context.Context, sprintID string, storyIDs []string) error {
	return r.record("", sprintID, storyIDs)
}
func (r *assignmentRepository) AssignStoriesForProject(_ context.Context, projectID string, sprintID string, storyIDs []string) error {
	return r.record(projectID, sprintID, storyIDs)
}
func (r *assignmentRepository) record(projectID string, sprintID string, storyIDs []string) error {
	r.calls++
	r.projectID = projectID
	r.sprintID = sprintID
	r.storyIDs = append([]string(nil), storyIDs...)
	return r.err
}

func TestAssignStoriesPassesWholeSingleOrBatchToRepository(t *testing.T) {
	for _, ids := range [][]string{{"story-1"}, {"story-1", "story-2"}} {
		t.Run(strings.Join(ids, ","), func(t *testing.T) {
			repo := &assignmentRepository{}
			useCase := application.NewAssignStoriesUseCase(repo)
			if err := useCase.Execute(context.Background(), application.AssignStoriesCommand{ProjectID: "project-1", SprintID: "sprint-1", StoryIDs: ids}); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if repo.calls != 1 || repo.projectID != "project-1" || repo.sprintID != "sprint-1" || !reflect.DeepEqual(repo.storyIDs, ids) {
				t.Errorf("repository call = %d, %q, %v; want one whole-batch call", repo.calls, repo.sprintID, repo.storyIDs)
			}
		})
	}
}

type unscopedAssignmentRepository struct{ calls int }

func (r *unscopedAssignmentRepository) AssignStories(context.Context, string, []string) error {
	r.calls++
	return nil
}
func TestAssignStoriesRejectsUnscopedAssignerWithoutWriting(t *testing.T) {
	repo := &unscopedAssignmentRepository{}
	err := application.NewAssignStoriesUseCase(repo).Execute(context.Background(), application.AssignStoriesCommand{
		ProjectID: "project-1", SprintID: "sprint-1", StoryIDs: []string{"story-1"},
	})
	if !errors.Is(err, application.ErrProjectScopedAssignerRequired) || repo.calls != 0 {
		t.Fatalf("error/writes = %v/%d, want rejection and zero writes", err, repo.calls)
	}
}
func TestAssignStoriesRejectsEmptyAndRepeatedIDsWithoutWriting(t *testing.T) {
	cases := []struct {
		name string
		ids  []string
		want error
	}{
		{"empty selection", nil, application.ErrEmptyStorySelection},
		{"repeated story", []string{"story-1", "story-2", "story-1"}, application.ErrDuplicateStoryID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &assignmentRepository{}
			err := application.NewAssignStoriesUseCase(repo).Execute(context.Background(), application.AssignStoriesCommand{ProjectID: "project-1", SprintID: "sprint-1", StoryIDs: tc.ids})
			if !errors.Is(err, tc.want) || repo.calls != 0 {
				t.Errorf("error/writes = %v/%d, want %v and zero writes", err, repo.calls, tc.want)
			}
		})
	}
}
func TestAssignStoriesPropagatesRepositoryRejectionsWithoutClaimingSuccess(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"sprint not found", application.ErrSprintNotFound},
		{"story not found", application.ErrStoryNotFound},
		{"different project", application.ErrProjectMismatch},
		{"already assigned", application.ErrStoryAlreadyAssigned},
		{"closed sprint", application.ErrSprintClosed},
		{"database failure", errors.New("database unavailable")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &assignmentRepository{err: tc.err}
			err := application.NewAssignStoriesUseCase(repo).Execute(context.Background(), application.AssignStoriesCommand{ProjectID: "project-1", SprintID: "sprint-1", StoryIDs: []string{"story-1", "story-2"}})
			if !errors.Is(err, tc.err) || repo.calls != 1 {
				t.Errorf("error/writes = %v/%d, want propagated rejection %v after one atomic repository call", err, repo.calls, tc.err)
			}
		})
	}
}
