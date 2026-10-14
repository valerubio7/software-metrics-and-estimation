package transporthttp

import (
	"errors"
	"net/http"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

type assignedStoriesResponse struct {
	ProjectID string                  `json:"project_id"`
	SprintID  string                  `json:"sprint_id"`
	Stories   []assignedStoryResponse `json:"stories"`
}

type assignedStoryResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Priority string `json:"priority"`
}

// NewListAssignedSprintStoriesHandler serves stories assigned to the project-scoped sprint.
func NewListAssignedSprintStoriesHandler(useCase *application.ListAssignedSprintStoriesUseCase) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writeJSON(response, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed", Message: "only GET is supported"})
			return
		}
		result, err := useCase.Execute(request.Context(), application.ListAssignedSprintStoriesQuery{
			ProjectID: request.PathValue("project_id"), SprintID: request.PathValue("sprint_id"),
		})
		if err != nil {
			var validation *domain.ValidationError
			switch {
			case errors.As(err, &validation):
				writeJSON(response, http.StatusUnprocessableEntity, errorResponse{Error: "validation_failed", Message: "one or more fields are invalid", Fields: validation.Fields})
			case errors.Is(err, application.ErrProjectNotFound):
				writeJSON(response, http.StatusNotFound, errorResponse{Error: "project_not_found", Message: "project not found"})
			case errors.Is(err, application.ErrSprintNotFound):
				writeJSON(response, http.StatusNotFound, errorResponse{Error: "sprint_not_found", Message: "sprint not found"})
			default:
				writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal_error", Message: "an unexpected error occurred"})
			}
			return
		}
		stories := make([]assignedStoryResponse, 0, len(result.Stories))
		for _, story := range result.Stories {
			stories = append(stories, assignedStoryResponse{ID: story.ID, Title: story.Title, Priority: story.Priority})
		}
		writeJSON(response, http.StatusOK, assignedStoriesResponse{ProjectID: result.ProjectID, SprintID: result.SprintID, Stories: stories})
	})
}
