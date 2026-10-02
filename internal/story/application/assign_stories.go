package application

import (
	"context"
	"errors"
)

var (
	// ErrEmptyStorySelection rejects assignment requests without stories.
	ErrEmptyStorySelection = errors.New("story selection must not be empty")
	// ErrDuplicateStoryID rejects repeated story identifiers in one request.
	ErrDuplicateStoryID = errors.New("story selection contains a duplicate ID")
	// ErrSprintNotFound identifies a sprint that does not exist.
	ErrSprintNotFound = errors.New("sprint not found")
	// ErrSprintClosed rejects planning work into a closed sprint.
	ErrSprintClosed = errors.New("sprint is closed")
	// ErrProjectMismatch identifies stories and a sprint from different projects.
	ErrProjectMismatch = errors.New("stories and sprint belong to different projects")
	// ErrStoryAlreadyAssigned identifies a story already linked to the selected sprint.
	ErrStoryAlreadyAssigned = errors.New("story already assigned to sprint")
	// ErrProjectScopedAssignerRequired rejects persistence ports that cannot enforce route project scope.
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

// ProjectScopedStorySprintAssigner verifies the project from the route alongside
// the sprint and every selected story in one atomic persistence operation.
type ProjectScopedStorySprintAssigner interface {
	AssignStoriesForProject(ctx context.Context, projectID string, sprintID string, storyIDs []string) error
}

// AssignStoriesUseCase validates request-level constraints before delegating the
// complete batch to a repository that enforces project scope and atomic persistence.
type AssignStoriesUseCase struct {
	assigner StorySprintAssigner
}

// NewAssignStoriesUseCase builds the story assignment use case.
func NewAssignStoriesUseCase(assigner StorySprintAssigner) *AssignStoriesUseCase {
	return &AssignStoriesUseCase{assigner: assigner}
}

// Execute rejects malformed selections before any persistence call. The repository
// checks resource existence, project membership, and prior associations atomically.
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
