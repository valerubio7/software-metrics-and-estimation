package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

func TestUpdateProjectUseCaseReplacesValidProject(t *testing.T) {
	tests := []struct {
		name       string
		finishDate string
	}{
		{name: "different dates", finishDate: "2026-06-30"},
		{name: "equal dates", finishDate: "2026-03-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			useCase := application.NewUpdateProjectUseCase(repository)
			command := validUpdateCommand()
			command.PlannedFinishDate = tt.finishDate
			project, err := useCase.Execute(context.Background(), command)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if project.ID != "existing-id" || project.Name != "Updated" || project.StartDate.Format("2006-01-02") != "2026-03-01" || project.PlannedFinishDate.Format("2006-01-02") != tt.finishDate {
				t.Errorf("updated project = %#v", project)
			}
			if repository.calls != 1 || len(repository.projects) != 1 || !sameProject(repository.projects[0], project) {
				t.Errorf("repository writes = %#v, want exactly updated project", repository.projects)
			}
		})
	}
}

func TestUpdateProjectUseCaseRejectsInvalidReplacementWithoutPersisting(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*application.UpdateProjectCommand)
		field  string
	}{
		{name: "missing name", mutate: func(c *application.UpdateProjectCommand) { c.Name = "" }, field: "name"},
		{name: "blank name", mutate: func(c *application.UpdateProjectCommand) { c.Name = "  " }, field: "name"},
		{name: "missing start date", mutate: func(c *application.UpdateProjectCommand) { c.StartDate = "" }, field: "start_date"},
		{name: "invalid start date", mutate: func(c *application.UpdateProjectCommand) { c.StartDate = "2026-02-30" }, field: "start_date"},
		{name: "missing finish date", mutate: func(c *application.UpdateProjectCommand) { c.PlannedFinishDate = "" }, field: "planned_finish_date"},
		{name: "invalid finish date", mutate: func(c *application.UpdateProjectCommand) { c.PlannedFinishDate = "not-a-date" }, field: "planned_finish_date"},
		{name: "finish before start", mutate: func(c *application.UpdateProjectCommand) { c.PlannedFinishDate = "2026-02-28" }, field: "planned_finish_date"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			useCase := application.NewUpdateProjectUseCase(repository)
			command := validUpdateCommand()
			tt.mutate(&command)
			_, err := useCase.Execute(context.Background(), command)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Fields[tt.field] == "" {
				t.Fatalf("Execute() error = %v, want validation error for %s", err, tt.field)
			}
			if repository.calls != 0 {
				t.Errorf("repository writes = %d, want 0", repository.calls)
			}
		})
	}
}

func TestUpdateProjectUseCasePropagatesRepositoryErrors(t *testing.T) {
	for name, repositoryError := range map[string]error{
		"database error": errors.New("database unavailable"),
		"not found":      application.ErrProjectNotFound,
	} {
		t.Run(name, func(t *testing.T) {
			repository := &fakeRepository{err: repositoryError}
			useCase := application.NewUpdateProjectUseCase(repository)
			_, err := useCase.Execute(context.Background(), validUpdateCommand())
			if !errors.Is(err, repositoryError) {
				t.Errorf("Execute() error = %v, want %v", err, repositoryError)
			}
		})
	}
}

func validUpdateCommand() application.UpdateProjectCommand {
	return application.UpdateProjectCommand{
		ProjectID: "existing-id", Name: "Updated", StartDate: "2026-03-01", PlannedFinishDate: "2026-06-30",
	}
}
