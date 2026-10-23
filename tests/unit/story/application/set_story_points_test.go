package application_test

import (
	"context"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
	"testing"
)

type pointsSetterSpy struct {
	calls              int
	projectID, storyID string
	points             int
	result             domain.Story
	err                error
}

func (s *pointsSetterSpy) SetStoryPoints(_ context.Context, projectID, storyID string, points int) (domain.Story, error) {
	s.calls++
	s.projectID = projectID
	s.storyID = storyID
	s.points = points
	return s.result, s.err
}
func TestSetStoryPointsPersistsFinalAgreedEstimate(t *testing.T) {
	spy := &pointsSetterSpy{result: domain.Story{ID: "00000000-0000-0000-0000-000000000002", ProjectID: "00000000-0000-0000-0000-000000000001"}}
	story, err := application.NewSetStoryPointsUseCase(spy).Execute(context.Background(), application.SetStoryPointsCommand{ProjectID: spy.result.ProjectID, StoryID: spy.result.ID, StoryPoints: 13})
	if err != nil || spy.calls != 1 || spy.points != 13 || story.ID != spy.result.ID {
		t.Fatalf("story=%+v calls=%d err=%v", story, spy.calls, err)
	}
}
func TestSetStoryPointsRejectsUndecidedAndUnsupportedValuesBeforeStorage(t *testing.T) {
	for _, points := range []int{-1, 4, 21, 101} {
		spy := &pointsSetterSpy{}
		_, err := application.NewSetStoryPointsUseCase(spy).Execute(context.Background(), application.SetStoryPointsCommand{ProjectID: "00000000-0000-0000-0000-000000000001", StoryID: "00000000-0000-0000-0000-000000000002", StoryPoints: points})
		if err == nil || spy.calls != 0 {
			t.Errorf("points=%d err=%v calls=%d", points, err, spy.calls)
		}
	}
}
