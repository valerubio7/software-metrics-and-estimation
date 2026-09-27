package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

type sprintRepository struct {
	calls  int
	stored domain.Sprint
	err    error
}

func (r *sprintRepository) Create(_ context.Context, sprint domain.Sprint) error {
	r.calls++
	r.stored = sprint
	return r.err
}

func validCommand() application.CreateSprintCommand {
	return application.CreateSprintCommand{
		ProjectID:  "project-id",
		SprintGoal: "  Deliver the initial metrics flow  ",
	}
}

func TestCreateSprintPersistsOnceAndReturnsServerGeneratedSprint(t *testing.T) {
	repo := &sprintRepository{}
	useCase := application.NewCreateSprintUseCase(repo, func() string { return "generated-id" })
	sprint, err := useCase.Execute(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	want := domain.Sprint{
		ID: "generated-id", ProjectID: "project-id", SprintGoal: "  Deliver the initial metrics flow  ",
	}
	if repo.calls != 1 || !reflect.DeepEqual(repo.stored, want) {
		t.Errorf("calls/stored = %d/%#v, want one write of %#v", repo.calls, repo.stored, want)
	}
	if !reflect.DeepEqual(sprint, want) {
		t.Errorf("result = %#v, want %#v", sprint, want)
	}
}

func TestCreateSprintRejectsInvalidGoalBeforeGeneratingIDOrWriting(t *testing.T) {
	tests := []struct {
		name string
		goal string
	}{
		{name: "missing goal"},
		{name: "whitespace-only goal", goal: " \t\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &sprintRepository{}
			generated := 0
			command := validCommand()
			command.SprintGoal = tt.goal
			useCase := application.NewCreateSprintUseCase(repo, func() string {
				generated++
				return "generated-id"
			})
			_, err := useCase.Execute(context.Background(), command)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Fields["sprint_goal"] == "" {
				t.Errorf("Execute() error = %v, want sprint_goal validation error", err)
			}
			if generated != 0 || repo.calls != 0 {
				t.Errorf("generated IDs/writes = %d/%d, want zero before valid input", generated, repo.calls)
			}
		})
	}
}

func TestCreateSprintPropagatesRepositoryErrorWithoutClaimingSuccess(t *testing.T) {
	repositoryError := errors.New("database unavailable")
	repo := &sprintRepository{err: repositoryError}
	sprint, err := application.NewCreateSprintUseCase(repo, func() string {
		return "generated-id"
	}).Execute(context.Background(), validCommand())
	if !errors.Is(err, repositoryError) || repo.calls != 1 || !reflect.DeepEqual(sprint, domain.Sprint{}) {
		t.Errorf("result/error/writes = %#v/%v/%d, want empty result, original error, one attempted write", sprint, err, repo.calls)
	}
}
