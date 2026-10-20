package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

type projectLister struct {
	projects []domain.Project
	err      error
	calls    int
}

func (l *projectLister) List(_ context.Context) ([]domain.Project, error) {
	l.calls++
	return l.projects, l.err
}

func TestListProjectsDerivesStatusForEveryProject(t *testing.T) {
	reference := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	lister := &projectLister{projects: []domain.Project{
		{ID: "p1", Name: "Activo", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), PlannedFinishDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)},
		{ID: "p2", Name: "Planificado", StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), PlannedFinishDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "p3", Name: "Atrasado", StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), PlannedFinishDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}}

	summaries, err := application.NewListProjectsUseCase(lister).Execute(context.Background(), reference)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if lister.calls != 1 {
		t.Fatalf("lister calls = %d, want 1", lister.calls)
	}
	if len(summaries) != 3 {
		t.Fatalf("summaries = %d, want 3", len(summaries))
	}
	want := map[string]domain.ProjectStatus{
		"p1": domain.ProjectStatusActive,
		"p2": domain.ProjectStatusPlanned,
		"p3": domain.ProjectStatusOverdue,
	}
	for _, summary := range summaries {
		if summary.Status != want[summary.Project.ID] {
			t.Errorf("project %s status = %q, want %q", summary.Project.ID, summary.Status, want[summary.Project.ID])
		}
	}
}

func TestListProjectsReturnsEmptyCollectionWithoutError(t *testing.T) {
	summaries, err := application.NewListProjectsUseCase(&projectLister{}).Execute(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if summaries == nil || len(summaries) != 0 {
		t.Fatalf("summaries = %#v, want non-nil empty collection", summaries)
	}
}

func TestListProjectsPropagatesPortError(t *testing.T) {
	sentinel := errors.New("connection refused")
	summaries, err := application.NewListProjectsUseCase(&projectLister{err: sentinel}).Execute(context.Background(), time.Now())
	if !errors.Is(err, sentinel) {
		t.Fatalf("Execute() error = %v, want %v", err, sentinel)
	}
	if summaries != nil {
		t.Fatalf("summaries = %#v, want nil on error", summaries)
	}
}
