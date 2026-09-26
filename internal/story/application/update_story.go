package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

// ErrStoryNotFound identifies a story that does not exist or belongs to another project.
var ErrStoryNotFound = errors.New("story not found")

// UpdateStoryCommand contains the full replacement of a story's editable content.
// MissingFields names the required keys that the input did not supply.
type UpdateStoryCommand struct {
	ProjectID          string
	StoryID            string
	Title              string
	Description        string
	Priority           string
	Status             string
	AcceptanceCriteria []string
	EstimatedHours     *float64
	MissingFields      []string
}

// StoryUpdater replaces the editable content of a project's story and returns the stored row.
type StoryUpdater interface {
	Update(ctx context.Context, story domain.Story) (domain.Story, error)
}

// UpdateStoryUseCase validates completely before a single write.
type UpdateStoryUseCase struct {
	updater StoryUpdater
}

// NewUpdateStoryUseCase builds an update-story use case with its only dependency.
func NewUpdateStoryUseCase(updater StoryUpdater) *UpdateStoryUseCase {
	return &UpdateStoryUseCase{updater: updater}
}

// Execute returns a *domain.ValidationError with every invalid field and without touching
// the port, ErrStoryNotFound when the story does not exist under that project, or the
// stored story.
func (u *UpdateStoryUseCase) Execute(ctx context.Context, command UpdateStoryCommand) (domain.Story, error) {
	fields := make(map[string]string)
	for _, field := range command.MissingFields {
		fields[field] = "is required"
	}
	projectID, err := uuid.Parse(command.ProjectID)
	if err != nil {
		fields["project_id"] = "must be a valid UUID"
	}
	storyID, err := uuid.Parse(command.StoryID)
	if err != nil {
		fields["story_id"] = "must be a valid UUID"
	}

	story, err := domain.NewStoryUpdate(storyID.String(), projectID.String(), command.Title, command.Description,
		command.Priority, command.Status, command.AcceptanceCriteria, command.EstimatedHours)
	var validation *domain.ValidationError
	if errors.As(err, &validation) {
		for field, message := range validation.Fields {
			// A missing key keeps its "is required" message over the zero-value content message.
			if _, taken := fields[field]; !taken {
				fields[field] = message
			}
		}
	}
	if len(fields) > 0 {
		return domain.Story{}, &domain.ValidationError{Fields: fields}
	}

	stored, err := u.updater.Update(ctx, story)
	if err != nil {
		return domain.Story{}, err
	}
	return stored, nil
}
