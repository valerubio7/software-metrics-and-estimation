package application

import (
	"context"
	"errors"
	"time"
)

var (
	ErrStoryNotInSprint      = errors.New("story is not assigned to sprint")
	ErrStoryAlreadyCompleted = errors.New("story already completed in sprint")
)

// SprintStoryCompletion is the recorded completion of one story within one sprint.
type SprintStoryCompletion struct {
	ProjectID   string
	SprintID    string
	StoryID     string
	CompletedAt time.Time
}

// CompleteSprintStoryCommand identifies the story to complete within the sprint.
type CompleteSprintStoryCommand struct {
	ProjectID string
	SprintID  string
	StoryID   string
}

// SprintStoryCompleter verifies, in one transaction, that the project exists, the sprint
// and the story belong to it, the sprint is open, the story is assigned to the sprint and
// not yet completed there, then records the completion instant. It returns
// ErrProjectNotFound, ErrSprintNotFound, ErrStoryNotFound, ErrSprintClosed,
// ErrStoryNotInSprint or ErrStoryAlreadyCompleted, in that precedence; other errors are
// returned unchanged.
type SprintStoryCompleter interface {
	CompleteSprintStory(ctx context.Context, projectID, sprintID, storyID string) (SprintStoryCompletion, error)
}

// CompleteSprintStoryUseCase records the completion of a story within a sprint.
type CompleteSprintStoryUseCase struct {
	completer SprintStoryCompleter
}

func NewCompleteSprintStoryUseCase(completer SprintStoryCompleter) *CompleteSprintStoryUseCase {
	return &CompleteSprintStoryUseCase{completer: completer}
}

// Execute delegates to the completer and returns its result unchanged (errors unwrapped,
// so errors.Is keeps working).
func (u *CompleteSprintStoryUseCase) Execute(ctx context.Context, command CompleteSprintStoryCommand) (SprintStoryCompletion, error) {
	return u.completer.CompleteSprintStory(ctx, command.ProjectID, command.SprintID, command.StoryID)
}
