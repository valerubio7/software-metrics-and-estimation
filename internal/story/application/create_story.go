package application

import (
	"context"
	"errors"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

// ErrProjectNotFound identifies a missing project when the repository inserts a story.
var ErrProjectNotFound = errors.New("project not found")

// CreateStoryCommand contains only values supplied to create a story.
type CreateStoryCommand struct {
	ProjectID          string
	Title              string
	Description        string
	Priority           string
	AcceptanceCriteria []string
}

// StoryRepository persists a validated story and ensures its project association.
type StoryRepository interface {
	Create(ctx context.Context, story domain.Story) error
}

// IDGenerator creates identifiers for new stories.
type IDGenerator func() string

// CreateStoryUseCase creates a pending story with one repository write.
type CreateStoryUseCase struct {
	repository StoryRepository
	generateID IDGenerator
}

// NewCreateStoryUseCase builds a create-story use case with its dependencies.
func NewCreateStoryUseCase(repository StoryRepository, generateID IDGenerator) *CreateStoryUseCase {
	return &CreateStoryUseCase{repository: repository, generateID: generateID}
}

// Execute validates and inserts a story, returning it only after persistence succeeds.
func (u *CreateStoryUseCase) Execute(ctx context.Context, command CreateStoryCommand) (domain.Story, error) {
	story, err := domain.NewStory("", command.ProjectID, command.Title, command.Description, command.Priority, command.AcceptanceCriteria)
	if err != nil {
		return domain.Story{}, err
	}
	story.ID = u.generateID()
	if err := u.repository.Create(ctx, story); err != nil {
		return domain.Story{}, err
	}
	return story, nil
}
