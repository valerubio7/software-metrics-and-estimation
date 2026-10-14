package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

// AssignedSprintStoryReader reads stories assigned to a project-scoped sprint.
type AssignedSprintStoryReader interface {
	ListAssignedToSprint(ctx context.Context, projectID, sprintID string) ([]domain.AssignedSprintStory, error)
}

// ListAssignedSprintStoriesQuery identifies the project and sprint for assigned-story discovery.
type ListAssignedSprintStoriesQuery struct {
	ProjectID string
	SprintID  string
}

// AssignedSprintStories is the project and sprint context plus assigned story results.
type AssignedSprintStories struct {
	ProjectID string
	SprintID  string
	Stories   []domain.AssignedSprintStory
}

// ListAssignedSprintStoriesUseCase reads only stories assigned to the selected sprint.
type ListAssignedSprintStoriesUseCase struct {
	reader AssignedSprintStoryReader
}

// NewListAssignedSprintStoriesUseCase builds an assigned-story listing use case.
func NewListAssignedSprintStoriesUseCase(reader AssignedSprintStoryReader) *ListAssignedSprintStoriesUseCase {
	return &ListAssignedSprintStoriesUseCase{reader: reader}
}

// Execute validates both route identifiers before reading the scoped assignments.
func (u *ListAssignedSprintStoriesUseCase) Execute(ctx context.Context, query ListAssignedSprintStoriesQuery) (AssignedSprintStories, error) {
	projectID, err := uuid.Parse(query.ProjectID)
	if err != nil {
		return AssignedSprintStories{}, &domain.ValidationError{Fields: map[string]string{"project_id": "must be a valid UUID"}}
	}
	sprintID, err := uuid.Parse(query.SprintID)
	if err != nil {
		return AssignedSprintStories{}, &domain.ValidationError{Fields: map[string]string{"sprint_id": "must be a valid UUID"}}
	}
	stories, err := u.reader.ListAssignedToSprint(ctx, projectID.String(), sprintID.String())
	if err != nil {
		return AssignedSprintStories{}, err
	}
	return AssignedSprintStories{ProjectID: projectID.String(), SprintID: sprintID.String(), Stories: stories}, nil
}

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
