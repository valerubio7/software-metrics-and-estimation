package transporthttp

import (
	"errors"
	"net/http"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

type listOpenSprintsResponse struct {
	ProjectID string               `json:"project_id"`
	Sprints   []openSprintResponse `json:"sprints"`
}

type openSprintResponse struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	SprintGoal string `json:"sprint_goal"`
	IsClosed   bool   `json:"is_closed"`
}

// NewListOpenSprintsHandler serves the project's open sprints without changing closed history.
func NewListOpenSprintsHandler(useCase *application.ListOpenSprintsUseCase) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writeJSON(response, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed", Message: "only GET is supported"})
			return
		}
		sprints, projectID, err := useCase.Execute(request.Context(), application.ListOpenSprintsQuery{ProjectID: request.PathValue("project_id")})
		if err != nil {
			var validation *domain.ValidationError
			switch {
			case errors.As(err, &validation):
				writeJSON(response, http.StatusUnprocessableEntity, errorResponse{Error: "validation_failed", Message: "one or more fields are invalid", Fields: validation.Fields})
			case errors.Is(err, application.ErrProjectNotFound):
				writeJSON(response, http.StatusNotFound, errorResponse{Error: "project_not_found", Message: "project not found"})
			default:
				writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal_error", Message: "an unexpected error occurred"})
			}
			return
		}
		result := make([]openSprintResponse, 0, len(sprints))
		for _, sprint := range sprints {
			result = append(result, openSprintResponse{ID: sprint.ID, ProjectID: sprint.ProjectID, SprintGoal: sprint.SprintGoal})
		}
		writeJSON(response, http.StatusOK, listOpenSprintsResponse{ProjectID: projectID, Sprints: result})
	})
}
