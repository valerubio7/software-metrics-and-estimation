package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

const (
	updateProjectID = "0f8fad5b-d9cb-469f-a165-70867728950e"
	updateStoryID   = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
)

// storyUpdater is a hand-written fake of the StoryUpdater port. It counts calls,
// captures the story it receives and returns a preconfigured result.
type storyUpdater struct {
	calls    int
	received domain.Story
	result   domain.Story
	err      error
}

func (u *storyUpdater) Update(_ context.Context, story domain.Story) (domain.Story, error) {
	u.calls++
	u.received = story
	return u.result, u.err
}

func hours(value float64) *float64 { return &value }

func validUpdateCommand() application.UpdateStoryCommand {
	return application.UpdateStoryCommand{
		ProjectID: updateProjectID, StoryID: updateStoryID,
		Title: " Título nuevo ", Description: "Descripción nueva", Priority: "alta",
		Status: "en_progreso", AcceptanceCriteria: []string{" Primero ", "Segundo"},
		EstimatedHours: hours(8.5),
	}
}

func assertValidationFields(t *testing.T, err error, want map[string]string) {
	t.Helper()
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want *domain.ValidationError", err)
	}
	if !reflect.DeepEqual(validation.Fields, want) {
		t.Errorf("fields = %#v, want %#v", validation.Fields, want)
	}
}

func TestUpdateStoryWritesOnceWithCanonicalIdentifiersAndAllSixFields(t *testing.T) {
	stored := domain.Story{ID: updateStoryID, ProjectID: updateProjectID, Title: "stored by the database"}
	updater := &storyUpdater{result: stored}
	command := validUpdateCommand()
	command.ProjectID = "0F8FAD5B-D9CB-469F-A165-70867728950E"
	command.StoryID = "7C9E6679-7425-40DE-944B-E07FC1F90AE7"

	story, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if updater.calls != 1 {
		t.Fatalf("updater calls = %d, want exactly one", updater.calls)
	}
	want := domain.Story{
		ID: updateStoryID, ProjectID: updateProjectID,
		Title: " Título nuevo ", Description: "Descripción nueva", Priority: "alta",
		Status: "en_progreso", AcceptanceCriteria: []string{" Primero ", "Segundo"},
		EstimatedHours: hours(8.5),
	}
	if !reflect.DeepEqual(updater.received, want) {
		t.Errorf("story sent to the port = %#v, want %#v", updater.received, want)
	}
	if !reflect.DeepEqual(story, stored) {
		t.Errorf("result = %#v, want the story returned by the port %#v", story, stored)
	}
}

func TestUpdateStoryDoesNotWriteInvalidInput(t *testing.T) {
	updater := &storyUpdater{}
	command := validUpdateCommand()
	command.Title = " \t"

	story, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), command)

	assertValidationFields(t, err, map[string]string{"title": "is required"})
	if updater.calls != 0 {
		t.Errorf("updater calls = %d, want zero", updater.calls)
	}
	if !reflect.DeepEqual(story, domain.Story{}) {
		t.Errorf("result = %#v, want the empty story", story)
	}
}

func TestUpdateStoryPropagatesPortErrorsWithoutClaimingSuccess(t *testing.T) {
	for _, failure := range []error{application.ErrStoryNotFound, errors.New("database unavailable")} {
		t.Run(failure.Error(), func(t *testing.T) {
			updater := &storyUpdater{result: domain.Story{ID: "must not leak"}, err: failure}

			story, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), validUpdateCommand())

			if !errors.Is(err, failure) || updater.calls != 1 || !reflect.DeepEqual(story, domain.Story{}) {
				t.Errorf("result/error/calls = %#v/%v/%d, want empty result, original error, one attempted write", story, err, updater.calls)
			}
		})
	}
}

func TestUpdateStoryErrStoryNotFoundIsDistinctFromProjectNotFound(t *testing.T) {
	if errors.Is(application.ErrStoryNotFound, application.ErrProjectNotFound) {
		t.Error("ErrStoryNotFound must not be confused with ErrProjectNotFound")
	}
}

var requiredUpdateFields = []string{"title", "description", "priority", "status", "acceptance_criteria", "estimated_hours"}

func TestUpdateStoryReportsEverySuppliedMissingFieldAsRequired(t *testing.T) {
	updater := &storyUpdater{}
	command := application.UpdateStoryCommand{ProjectID: updateProjectID, StoryID: updateStoryID, MissingFields: requiredUpdateFields}

	_, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), command)

	assertValidationFields(t, err, map[string]string{
		"title": "is required", "description": "is required", "priority": "is required",
		"status": "is required", "acceptance_criteria": "is required", "estimated_hours": "is required",
	})
	if updater.calls != 0 {
		t.Errorf("updater calls = %d, want zero", updater.calls)
	}
}

func TestUpdateStoryReportsASingleMissingFieldOnly(t *testing.T) {
	for _, field := range requiredUpdateFields {
		t.Run(field, func(t *testing.T) {
			updater := &storyUpdater{}
			command := validUpdateCommand()
			command.MissingFields = []string{field}

			_, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), command)

			assertValidationFields(t, err, map[string]string{field: "is required"})
			if updater.calls != 0 {
				t.Errorf("updater calls = %d, want zero", updater.calls)
			}
		})
	}
}

func TestUpdateStoryMergesInvalidIdentifiersWithContentErrors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*application.UpdateStoryCommand)
		want   map[string]string
	}{
		{"invalid story id and empty title",
			func(c *application.UpdateStoryCommand) { c.StoryID = "abc"; c.Title = "" },
			map[string]string{"story_id": "must be a valid UUID", "title": "is required"}},
		{"both identifiers invalid",
			func(c *application.UpdateStoryCommand) { c.ProjectID = "no-es-uuid"; c.StoryID = "" },
			map[string]string{"project_id": "must be a valid UUID", "story_id": "must be a valid UUID"}},
		{"invalid project id, missing key and invalid status",
			func(c *application.UpdateStoryCommand) {
				c.ProjectID = "x"
				c.Status = "cancelada"
				c.MissingFields = []string{"estimated_hours"}
			},
			map[string]string{
				"project_id":      "must be a valid UUID",
				"status":          "must be pendiente, en_progreso or completada",
				"estimated_hours": "is required",
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			updater := &storyUpdater{}
			command := validUpdateCommand()
			tc.mutate(&command)

			_, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), command)

			assertValidationFields(t, err, tc.want)
			if updater.calls != 0 {
				t.Errorf("updater calls = %d, want zero", updater.calls)
			}
		})
	}
}

func TestUpdateStoryMissingKeyMessageWinsOverZeroValueContentMessage(t *testing.T) {
	// Zero values whose content message differs from "is required": if the missing-key
	// message were overwritten by the domain message, the assertion below would fail.
	updater := &storyUpdater{}
	command := validUpdateCommand()
	command.Priority = ""
	command.Status = ""
	command.AcceptanceCriteria = nil
	command.EstimatedHours = hours(0)
	command.MissingFields = []string{"priority", "status", "acceptance_criteria", "estimated_hours"}

	_, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), command)

	assertValidationFields(t, err, map[string]string{
		"priority": "is required", "status": "is required",
		"acceptance_criteria": "is required", "estimated_hours": "is required",
	})
	if updater.calls != 0 {
		t.Errorf("updater calls = %d, want zero", updater.calls)
	}
}

func TestUpdateStoryDistinguishesAbsentEstimatedHoursFromExplicitNull(t *testing.T) {
	t.Run("absent key is required", func(t *testing.T) {
		updater := &storyUpdater{}
		command := validUpdateCommand()
		command.EstimatedHours = nil
		command.MissingFields = []string{"estimated_hours"}

		_, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), command)

		assertValidationFields(t, err, map[string]string{"estimated_hours": "is required"})
		if updater.calls != 0 {
			t.Errorf("updater calls = %d, want zero", updater.calls)
		}
	})
	t.Run("present null clears the estimate", func(t *testing.T) {
		updater := &storyUpdater{result: domain.Story{ID: updateStoryID}}
		command := validUpdateCommand()
		command.EstimatedHours = nil

		_, err := application.NewUpdateStoryUseCase(updater).Execute(context.Background(), command)

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if updater.calls != 1 || updater.received.EstimatedHours != nil {
			t.Errorf("calls/estimated hours = %d/%v, want one write with a nil estimate", updater.calls, updater.received.EstimatedHours)
		}
	})
}
