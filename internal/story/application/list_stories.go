package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

// ListStoriesQuery identifies the project whose Product Backlog is requested.
type ListStoriesQuery struct {
	ProjectID string
}

// StoryLister reads the stories of an existing project in ascending creation order.
// It returns ErrProjectNotFound when the project does not exist; a project without
// stories is not an error.
type StoryLister interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Story, error)
}

// ListStoriesUseCase reads a project's Product Backlog without modifying anything.
type ListStoriesUseCase struct {
	lister StoryLister
}

// NewListStoriesUseCase builds a list-stories use case with its only dependency.
func NewListStoriesUseCase(lister StoryLister) *ListStoriesUseCase {
	return &ListStoriesUseCase{lister: lister}
}

// Execute returns a *domain.ValidationError without touching the port when project_id is
// not a UUID, the port's error without a partial list, or the Backlog ordered by priority
// under the canonical project identifier.
func (u *ListStoriesUseCase) Execute(ctx context.Context, query ListStoriesQuery) (domain.Backlog, error) {
	projectID, err := uuid.Parse(query.ProjectID)
	if err != nil {
		return domain.Backlog{}, &domain.ValidationError{Fields: map[string]string{"project_id": "must be a valid UUID"}}
	}

	stories, err := u.lister.ListByProject(ctx, projectID.String())
	if err != nil {
		return domain.Backlog{}, err
	}
	return domain.NewBacklog(projectID.String(), stories), nil
}
