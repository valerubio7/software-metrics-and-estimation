package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

type storyRepository struct {
	calls  int
	stored domain.Story
	err    error
}

func (r *storyRepository) Create(_ context.Context, story domain.Story) error {
	r.calls++
	r.stored = story
	return r.err
}

func validCommand() application.CreateStoryCommand {
	return application.CreateStoryCommand{
		ProjectID: "project-id", Title: " Título ", Description: "Descripción",
		Priority: "media", AcceptanceCriteria: []string{" Primero ", "Segundo"},
	}
}

func TestCreateStoryPersistsOnceAndReturnsGeneratedStory(t *testing.T) {
	repo := &storyRepository{}
	useCase := application.NewCreateStoryUseCase(repo, func() string { return "generated-id" })
	story, err := useCase.Execute(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repo.calls != 1 || !reflect.DeepEqual(repo.stored, story) {
		t.Errorf("calls/stored = %d/%#v, want one write of %#v", repo.calls, repo.stored, story)
	}
	if story.ID != "generated-id" || story.ProjectID != "project-id" || story.Title != " Título " || story.Description != "Descripción" || story.Priority != "media" {
		t.Errorf("result = %#v, want generated ID and unchanged input", story)
	}
	if story.Status != "pendiente" || story.StoryPoints != nil || !reflect.DeepEqual(story.AcceptanceCriteria, []string{" Primero ", "Segundo"}) {
		t.Errorf("initial state/criteria = %#v, want pending, nil points and ordered criteria", story)
	}
	if story.EstimatedHours != nil || repo.stored.EstimatedHours != nil {
		t.Errorf("estimated hours = %v/%v, want nil in the returned and persisted story", story.EstimatedHours, repo.stored.EstimatedHours)
	}
}

func TestCreateStoryDoesNotWriteInvalidInput(t *testing.T) {
	cases := []struct {
		name, field string
		mutate      func(*application.CreateStoryCommand)
	}{
		{"blank title", "title", func(c *application.CreateStoryCommand) { c.Title = " \t" }},
		{"blank description", "description", func(c *application.CreateStoryCommand) { c.Description = "\u2003" }},
		{"invalid priority", "priority", func(c *application.CreateStoryCommand) { c.Priority = "ALTA" }},
		{"missing criteria", "acceptance_criteria", func(c *application.CreateStoryCommand) { c.AcceptanceCriteria = nil }},
		{"blank criterion", "acceptance_criteria", func(c *application.CreateStoryCommand) { c.AcceptanceCriteria = []string{"OK", " "} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &storyRepository{}
			command := validCommand()
			tc.mutate(&command)
			_, err := application.NewCreateStoryUseCase(repo, func() string { return "generated-id" }).Execute(context.Background(), command)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Fields[tc.field] == "" {
				t.Errorf("error = %v, want validation of %q", err, tc.field)
			}
			if repo.calls != 0 {
				t.Errorf("writes = %d, want zero", repo.calls)
			}
		})
	}
}

func TestCreateStoryDoesNotGenerateIDForInvalidInput(t *testing.T) {
	repo := &storyRepository{}
	generated := 0
	useCase := application.NewCreateStoryUseCase(repo, func() string {
		generated++
		return "generated-id"
	})
	command := validCommand()
	command.AcceptanceCriteria = []string{"OK", "\u2003\u00a0"}
	_, err := useCase.Execute(context.Background(), command)
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || validation.Fields["acceptance_criteria"] == "" {
		t.Fatalf("error = %v, want invalid acceptance_criteria", err)
	}
	if generated != 0 || repo.calls != 0 {
		t.Errorf("generated IDs/writes = %d/%d, want zero before valid input", generated, repo.calls)
	}
}

func TestCreateStoryPropagatesRepositoryErrorsWithoutClaimingSuccess(t *testing.T) {
	for _, failure := range []error{application.ErrProjectNotFound, errors.New("database unavailable")} {
		t.Run(failure.Error(), func(t *testing.T) {
			repo := &storyRepository{err: failure}
			story, err := application.NewCreateStoryUseCase(repo, func() string { return "generated-id" }).Execute(context.Background(), validCommand())
			if !errors.Is(err, failure) || repo.calls != 1 || !reflect.DeepEqual(story, domain.Story{}) {
				t.Errorf("result/error/writes = %#v/%v/%d, want empty result, original error, one attempted write", story, err, repo.calls)
			}
		})
	}
}
