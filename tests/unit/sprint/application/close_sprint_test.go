package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
)

type sprintCloser struct {
	calls     int
	projectID string
	sprintID  string
	err       error
}

func (c *sprintCloser) CloseSprint(_ context.Context, projectID, sprintID string) error {
	c.calls++
	c.projectID = projectID
	c.sprintID = sprintID
	return c.err
}

func TestCloseSprintUseCaseDelegatesAndPropagatesResult(t *testing.T) {
	closeErr := errors.New("sprint not found")
	closer := &sprintCloser{err: closeErr}
	useCase := application.NewCloseSprintUseCase(closer)
	command := application.CloseSprintCommand{ProjectID: "project-id", SprintID: "sprint-id"}

	err := useCase.Execute(context.Background(), command)
	if !errors.Is(err, closeErr) || closer.calls != 1 || !reflect.DeepEqual(
		[]string{closer.projectID, closer.sprintID}, []string{command.ProjectID, command.SprintID},
	) {
		t.Fatalf("Execute() error/calls/IDs = %v/%d/%q/%q, want original error, one call and command IDs", err, closer.calls, closer.projectID, closer.sprintID)
	}
}
