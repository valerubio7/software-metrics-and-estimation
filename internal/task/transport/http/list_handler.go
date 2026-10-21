package transporthttp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
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

type listTasksResponse struct {
	ProjectID string         `json:"project_id"`
	StoryID   string         `json:"story_id"`
	Tasks     []taskResponse `json:"tasks"`
}

func (h *ListTasksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// A GET route also serves HEAD, so both methods must reach the handler.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only GET is supported",
		})
		return
	}

	fields := make(map[string]string)
	projectID, err := uuid.Parse(r.PathValue("project_id"))
	if err != nil {
		fields["project_id"] = "must be a valid UUID"
	}
	storyID, err := uuid.Parse(r.PathValue("story_id"))
	if err != nil {
		fields["story_id"] = "must be a valid UUID"
	}
	if len(fields) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
			Error: "validation_failed", Message: "one or more fields are invalid", Fields: fields,
		})
		return
	}

	result, err := h.useCase.Execute(r.Context(), application.ListTasksQuery{
		ProjectID: projectID.String(), StoryID: storyID.String(),
	})
	if err != nil {
		var validation *domain.ValidationError
		switch {
		case errors.As(err, &validation):
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
				Error: "validation_failed", Message: "one or more fields are invalid", Fields: validation.Fields,
			})
		case errors.Is(err, application.ErrProjectNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "project_not_found", Message: "project not found",
			})
		case errors.Is(err, application.ErrStoryNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "story_not_found", Message: "story not found",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "internal_error", Message: "an unexpected error occurred",
			})
		}
		return
	}

	tasks := make([]taskResponse, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		tasks = append(tasks, taskResponse{
			ID: task.ID, ProjectID: task.ProjectID, SprintID: task.SprintID, StoryID: task.StoryID,
			Title: task.Title, EstimatedHours: task.EstimatedHours,
		})
	}
	writeJSON(w, http.StatusOK, listTasksResponse{
		ProjectID: result.ProjectID, StoryID: result.StoryID, Tasks: tasks,
	})
}
