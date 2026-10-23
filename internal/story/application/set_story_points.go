package application

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

var ErrInvalidStoryPoints = errors.New("invalid story points")

type StoryPointsSetter interface {
	SetStoryPoints(context.Context, string, string, int) (domain.Story, error)
}
type SetStoryPointsCommand struct {
	ProjectID, StoryID string
	StoryPoints        int
}
type SetStoryPointsUseCase struct{ setter StoryPointsSetter }

func NewSetStoryPointsUseCase(setter StoryPointsSetter) *SetStoryPointsUseCase {
	return &SetStoryPointsUseCase{setter: setter}
}
func (u *SetStoryPointsUseCase) Execute(ctx context.Context, command SetStoryPointsCommand) (domain.Story, error) {
	projectID, projectErr := uuid.Parse(command.ProjectID)
	storyID, storyErr := uuid.Parse(command.StoryID)
	if projectErr != nil || storyErr != nil {
		fields := map[string]string{}
		if projectErr != nil {
			fields["project_id"] = "must be a valid UUID"
		}
		if storyErr != nil {
			fields["story_id"] = "must be a valid UUID"
		}
		return domain.Story{}, &domain.ValidationError{Fields: fields}
	}
	allowed := map[int]bool{0: true, 1: true, 2: true, 3: true, 5: true, 8: true, 13: true, 20: true, 40: true, 100: true}
	if !allowed[command.StoryPoints] {
		return domain.Story{}, &domain.ValidationError{Fields: map[string]string{"story_points": "must be one of 0, 1, 2, 3, 5, 8, 13, 20, 40 or 100"}}
	}
	return u.setter.SetStoryPoints(ctx, projectID.String(), storyID.String(), command.StoryPoints)
}
