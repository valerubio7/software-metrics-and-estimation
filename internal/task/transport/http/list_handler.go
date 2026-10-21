package transporthttp

import (
	"context"
	"net/http"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
)

type listTasks interface {
	Execute(context.Context, application.ListTasksQuery) (application.ProjectStoryTasks, error)
}

// ListTasksHandler serves the tasks of a project-scoped story.
type ListTasksHandler struct {
	useCase listTasks
}

// NewListTasksHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewListTasksHandler(useCase listTasks) http.Handler {
	return &ListTasksHandler{useCase: useCase}
}

func (h *ListTasksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
