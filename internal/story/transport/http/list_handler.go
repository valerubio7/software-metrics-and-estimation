package transporthttp

import (
	"errors"
	"net/http"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

// ListStoriesHandler serves the Product Backlog of the project identified by the route.
type ListStoriesHandler struct {
	useCase *application.ListStoriesUseCase
}

// NewListStoriesHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewListStoriesHandler(useCase *application.ListStoriesUseCase) http.Handler {
	return &ListStoriesHandler{useCase: useCase}
}

// backlogResponse is the container of the backlog, so paging metadata can be added later.
type backlogResponse struct {
	ProjectID string          `json:"project_id"`
	Stories   []storyResponse `json:"stories"`
}

func (h *ListStoriesHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// A GET route also serves HEAD, so both methods must reach the handler.
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only GET is supported",
		})
		return
	}

	backlog, err := h.useCase.Execute(request.Context(), application.ListStoriesQuery{
		ProjectID: request.PathValue("project_id"),
	})
	if err != nil {
		var validation *domain.ValidationError
		switch {
		case errors.As(err, &validation):
			writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
				Error: "validation_failed", Message: "one or more fields are invalid", Fields: validation.Fields,
			})
		case errors.Is(err, application.ErrProjectNotFound):
			writeJSON(response, http.StatusNotFound, errorResponse{
				Error: "project_not_found", Message: "project not found",
			})
		default:
			writeJSON(response, http.StatusInternalServerError, errorResponse{
				Error: "internal_error", Message: "an unexpected error occurred",
			})
		}
		return
	}

	// A non-nil slice serializes as [] even when the backlog has no stories.
	stories := make([]storyResponse, 0, len(backlog.Stories))
	for _, story := range backlog.Stories {
		stories = append(stories, newStoryResponse(story))
	}
	writeJSON(response, http.StatusOK, backlogResponse{ProjectID: backlog.ProjectID, Stories: stories})
}
