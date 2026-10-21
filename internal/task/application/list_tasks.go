package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

// TaskLister reads the tasks of a project-scoped story in creation order.
type TaskLister interface {
	ListByProjectStory(ctx context.Context, projectID, storyID string) ([]domain.Task, error)
}

// ListTasksQuery identifies the project and story whose tasks are requested.
type ListTasksQuery struct {
	ProjectID string
	StoryID   string
}

// ProjectStoryTasks is the route context plus the task collection.
type ProjectStoryTasks struct {
	ProjectID string
	StoryID   string
	Tasks     []domain.Task
}

// ListTasksUseCase reads a story's tasks without modifying anything.
type ListTasksUseCase struct {
	lister TaskLister
}

// NewListTasksUseCase builds a list-tasks use case with its only dependency.
func NewListTasksUseCase(lister TaskLister) *ListTasksUseCase {
	return &ListTasksUseCase{lister: lister}
}

// Execute validates both route identifiers before reading, returns the port error without
// a partial collection, and always yields a non-nil task slice.
func (u *ListTasksUseCase) Execute(ctx context.Context, query ListTasksQuery) (ProjectStoryTasks, error) {
	fields := make(map[string]string)
	projectID, err := uuid.Parse(query.ProjectID)
	if err != nil {
		fields["project_id"] = "must be a valid UUID"
	}
	storyID, err := uuid.Parse(query.StoryID)
	if err != nil {
		fields["story_id"] = "must be a valid UUID"
	}
	if len(fields) > 0 {
		return ProjectStoryTasks{}, &domain.ValidationError{Fields: fields}
	}

	tasks, err := u.lister.ListByProjectStory(ctx, projectID.String(), storyID.String())
	if err != nil {
		return ProjectStoryTasks{}, err
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	return ProjectStoryTasks{ProjectID: projectID.String(), StoryID: storyID.String(), Tasks: tasks}, nil
}
