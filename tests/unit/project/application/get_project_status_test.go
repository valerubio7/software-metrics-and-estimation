package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

type projectReader struct {
	project domain.Project
	err     error
	id      string
	calls   int
}

func (r *projectReader) GetByID(_ context.Context, id string) (domain.Project, error) {
	r.calls++
	r.id = id
	return r.project, r.err
}

func TestGetProjectStatusReadsIdentityAndDerivesStatus(t *testing.T) {
	reader := &projectReader{project: domain.Project{ID: "id", Name: "A", StartDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), PlannedFinishDate: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)}}
	useCase := application.NewGetProjectStatusUseCase(reader)
	project, status, err := useCase.Execute(context.Background(), "id", time.Date(2026, 3, 3, 22, 0, 0, 0, time.UTC))
	if err != nil || project.Name != "A" || status != domain.ProjectStatusActive || reader.calls != 1 || reader.id != "id" {
		t.Fatalf("got project=%+v status=%q calls=%d id=%q err=%v", project, status, reader.calls, reader.id, err)
	}
}

func TestGetProjectStatusPropagatesMissingWithoutStatus(t *testing.T) {
	reader := &projectReader{err: application.ErrProjectNotFound}
	_, status, err := application.NewGetProjectStatusUseCase(reader).Execute(context.Background(), "missing", time.Now())
	if !errors.Is(err, application.ErrProjectNotFound) || status != "" {
		t.Fatalf("status=%q err=%v", status, err)
	}
}
