package application

import (
	"context"
	"errors"
)

var (
	ErrEmptyStorySelection           = errors.New("story selection must not be empty")
	ErrDuplicateStoryID              = errors.New("story selection contains a duplicate ID")
	ErrSprintNotFound                = errors.New("sprint not found")
	ErrSprintClosed                  = errors.New("sprint is closed")
	ErrProjectMismatch               = errors.New("stories and sprint belong to different projects")
	ErrStoryAlreadyAssigned          = errors.New("story already assigned to sprint")
	ErrProjectScopedAssignerRequired = errors.New("project-scoped story assignment is required")
)

// AssignStoriesCommand identifies the sprint and complete story selection to assign.
type AssignStoriesCommand struct {
	ProjectID string
	SprintID  string
	StoryIDs  []string
}

// StorySprintAssigner persists a complete assignment batch atomically.
type StorySprintAssigner interface {
	AssignStories(ctx context.Context, sprintID string, storyIDs []string) error
}

// ProjectScopedStorySprintAssigner verifies route project, sprint and stories atomically.
type ProjectScopedStorySprintAssigner interface {
	AssignStoriesForProject(ctx context.Context, projectID string, sprintID string, storyIDs []string) error
}

// AssignStoriesUseCase validates selection before project-scoped persistence.
type AssignStoriesUseCase struct {
	assigner StorySprintAssigner
}

func NewAssignStoriesUseCase(assigner StorySprintAssigner) *AssignStoriesUseCase {
	return &AssignStoriesUseCase{assigner: assigner}
}

func (u *AssignStoriesUseCase) Execute(ctx context.Context, command AssignStoriesCommand) error {
	if len(command.StoryIDs) == 0 {
		return ErrEmptyStorySelection
	}
	seen := make(map[string]struct{}, len(command.StoryIDs))
	for _, storyID := range command.StoryIDs {
		if _, exists := seen[storyID]; exists {
			return ErrDuplicateStoryID
		}
		seen[storyID] = struct{}{}
	}
	ids := append([]string(nil), command.StoryIDs...)
	assigner, ok := u.assigner.(ProjectScopedStorySprintAssigner)
	if !ok {
		return ErrProjectScopedAssignerRequired
	}
	return assigner.AssignStoriesForProject(ctx, command.ProjectID, command.SprintID, ids)
}
