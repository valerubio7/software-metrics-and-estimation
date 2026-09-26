package transporthttp

import (
	"errors"
	"net/http"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

// requiredUpdateFields lists the keys a full replacement must carry, in report order.
var requiredUpdateFields = []string{"title", "description", "priority", "status", "acceptance_criteria", "estimated_hours"}

// UpdateStoryHandler handles the full replacement of a story identified by the route.
type UpdateStoryHandler struct {
	useCase *application.UpdateStoryUseCase
}

// NewUpdateStoryHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewUpdateStoryHandler(useCase *application.UpdateStoryUseCase) http.Handler {
	return &UpdateStoryHandler{useCase: useCase}
}

type updateStoryRequest struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EstimatedHours     *float64 `json:"estimated_hours"`
}

func (h *UpdateStoryHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPut {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only PUT is supported",
		})
		return
	}

	var input updateStoryRequest
	keys, err := decodeStoryObject(request.Body, &input)
	if err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Error: "invalid_request", Message: "request body must be a single valid JSON object with allowed fields",
		})
		return
	}

	// Only the raw key set tells an absent estimated_hours apart from an explicit null.
	var missing []string
	for _, field := range requiredUpdateFields {
		if _, present := keys[field]; !present {
			missing = append(missing, field)
		}
	}

	story, err := h.useCase.Execute(request.Context(), application.UpdateStoryCommand{
		ProjectID: request.PathValue("project_id"), StoryID: request.PathValue("story_id"),
		Title: input.Title, Description: input.Description, Priority: input.Priority, Status: input.Status,
		AcceptanceCriteria: input.AcceptanceCriteria, EstimatedHours: input.EstimatedHours,
		MissingFields: missing,
	})
	if err != nil {
		var validation *domain.ValidationError
		switch {
		case errors.As(err, &validation):
			writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
				Error: "validation_failed", Message: "one or more fields are invalid", Fields: validation.Fields,
			})
		case errors.Is(err, application.ErrStoryNotFound):
			writeJSON(response, http.StatusNotFound, errorResponse{
				Error: "story_not_found", Message: "story not found",
			})
		default:
			writeJSON(response, http.StatusInternalServerError, errorResponse{
				Error: "internal_error", Message: "an unexpected error occurred",
			})
		}
		return
	}

	writeJSON(response, http.StatusOK, newStoryResponse(story))
}
