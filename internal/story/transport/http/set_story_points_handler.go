package transporthttp

import (
	"encoding/json"
	"errors"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
	"net/http"
)

type SetStoryPointsHandler struct {
	useCase *application.SetStoryPointsUseCase
}

func NewSetStoryPointsHandler(useCase *application.SetStoryPointsUseCase) http.Handler {
	return &SetStoryPointsHandler{useCase: useCase}
}
func (h *SetStoryPointsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed", Message: "only PUT is supported"})
		return
	}
	var input struct {
		StoryPoints *int `json:"story_points"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || input.StoryPoints == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_request", Message: "request body must contain a numeric story_points field"})
		return
	}
	story, err := h.useCase.Execute(r.Context(), application.SetStoryPointsCommand{ProjectID: r.PathValue("project_id"), StoryID: r.PathValue("story_id"), StoryPoints: *input.StoryPoints})
	if err != nil {
		var validation *domain.ValidationError
		switch {
		case errors.As(err, &validation):
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "validation_failed", Message: "one or more fields are invalid", Fields: validation.Fields})
		case errors.Is(err, application.ErrStoryNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "story_not_found", Message: "story not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal_error", Message: "an unexpected error occurred"})
		}
		return
	}
	writeJSON(w, http.StatusOK, newStoryResponse(story))
}
