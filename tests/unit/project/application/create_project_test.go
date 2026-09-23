package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/valerubio7/software-metrics-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-estimation/internal/project/domain"
)

type fakeRepository struct {
	calls    int
	projects []domain.Project
	err      error
}

func (r *fakeRepository) Create(_ context.Context, project domain.Project) error {
	r.calls++
	r.projects = append(r.projects, project)
	return r.err
}

func TestCreateProjectUseCaseCreatesProjectAndPersistsIt(t *testing.T) {
	repository := &fakeRepository{}
	useCase := application.NewCreateProjectUseCase(repository, func() string { return "generated-id" })

	project, err := useCase.Execute(context.Background(), validCommand())

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if project.ID != "generated-id" {
		t.Errorf("project.ID = %q, want generated id", project.ID)
	}
	if project.Name != "Metrics portal" {
		t.Errorf("project.Name = %q, want %q", project.Name, "Metrics portal")
	}
	if repository.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.calls)
	}
	if got := repository.projects[0]; !sameProject(got, project) {
		t.Errorf("persisted project = %#v, want %#v", got, project)
	}
}

func TestCreateProjectUseCaseRejectsInvalidCommandsWithoutPersisting(t *testing.T) {
	tests := []struct {
		name  string
		mutate func(*application.CreateProjectCommand)
		field string
	}{
		{name: "missing name", mutate: func(c *application.CreateProjectCommand) { c.Name = "" }, field: "name"},
		{name: "blank name", mutate: func(c *application.CreateProjectCommand) { c.Name = " \t" }, field: "name"},
		{name: "missing start date", mutate: func(c *application.CreateProjectCommand) { c.StartDate = "" }, field: "start_date"},
		{name: "invalid start date", mutate: func(c *application.CreateProjectCommand) { c.StartDate = "2026-13-01" }, field: "start_date"},
		{name: "missing finish date", mutate: func(c *application.CreateProjectCommand) { c.PlannedFinishDate = "" }, field: "planned_finish_date"},
		{name: "invalid finish date", mutate: func(c *application.CreateProjectCommand) { c.PlannedFinishDate = "tomorrow" }, field: "planned_finish_date"},
		{name: "finish before start", mutate: func(c *application.CreateProjectCommand) { c.PlannedFinishDate = "2026-02-28" }, field: "planned_finish_date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			useCase := application.NewCreateProjectUseCase(repository, func() string { return "generated-id" })
			command := validCommand()
			tt.mutate(&command)

			_, err := useCase.Execute(context.Background(), command)

			var validationErr *domain.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("Execute() error = %v, want ValidationError", err)
			}
			if _, ok := validationErr.Fields[tt.field]; !ok {
				t.Errorf("validation fields = %#v, want %q", validationErr.Fields, tt.field)
			}
			if repository.calls != 0 {
				t.Errorf("repository calls = %d, want 0", repository.calls)
			}
		})
	}
}

func TestCreateProjectUseCaseAcceptsEqualDates(t *testing.T) {
	repository := &fakeRepository{}
	useCase := application.NewCreateProjectUseCase(repository, func() string { return "generated-id" })
	command := validCommand()
	command.PlannedFinishDate = command.StartDate

	_, err := useCase.Execute(context.Background(), command)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.calls != 1 {
		t.Errorf("repository calls = %d, want 1", repository.calls)
	}
}

func TestCreateProjectUseCasePropagatesRepositoryErrors(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	repository := &fakeRepository{err: repositoryErr}
	useCase := application.NewCreateProjectUseCase(repository, func() string { return "generated-id" })

	_, err := useCase.Execute(context.Background(), validCommand())

	if !errors.Is(err, repositoryErr) {
		t.Errorf("Execute() error = %v, want repository error", err)
	}
}

func validCommand() application.CreateProjectCommand {
	return application.CreateProjectCommand{
		Name:               "Metrics portal",
		StartDate:          "2026-03-01",
		PlannedFinishDate: "2026-06-30",
	}
}

func sameProject(left, right domain.Project) bool {
	return left.ID == right.ID &&
		left.Name == right.Name &&
		left.StartDate.Equal(right.StartDate) &&
		left.PlannedFinishDate.Equal(right.PlannedFinishDate)
}

