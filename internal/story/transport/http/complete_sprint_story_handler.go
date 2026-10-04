package transporthttp

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
)

type CompleteSprintStoryHandler struct {
	useCase *application.CompleteSprintStoryUseCase
}

func NewCompleteSprintStoryHandler(useCase *application.CompleteSprintStoryUseCase) http.Handler {
	return &CompleteSprintStoryHandler{useCase: useCase}
}

type sprintStoryCompletionResponse struct {
	ProjectID   string    `json:"project_id"`
	SprintID    string    `json:"sprint_id"`
	StoryID     string    `json:"story_id"`
	CompletedAt time.Time `json:"completed_at"`
}

// ServeHTTP never reads the request body: any content is ignored on purpose.
func (h *CompleteSprintStoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed", Message: "only POST is supported"})
		return
	}
	// Unlike the other story routes, every invalid route ID is reported at once.
	fields := map[string]string{}
	ids := map[string]string{}
	for _, name := range []string{"project_id", "sprint_id", "story_id"} {
		id, err := uuid.Parse(r.PathValue(name))
		if err != nil {
			fields[name] = "must be a valid UUID"
			continue
		}
		ids[name] = id.String()
	}
	if len(fields) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "validation_failed", Message: "one or more fields are invalid", Fields: fields})
		return
	}
	completion, err := h.useCase.Execute(r.Context(), application.CompleteSprintStoryCommand{ProjectID: ids["project_id"], SprintID: ids["sprint_id"], StoryID: ids["story_id"]})
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, sprintStoryCompletionResponse{ProjectID: completion.ProjectID, SprintID: completion.SprintID, StoryID: completion.StoryID, CompletedAt: completion.CompletedAt.UTC()})
	case errors.Is(err, application.ErrProjectNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "project_not_found", Message: "project not found"})
	case errors.Is(err, application.ErrSprintNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "sprint_not_found", Message: "sprint not found"})
	case errors.Is(err, application.ErrStoryNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "story_not_found", Message: "story not found"})
	case errors.Is(err, application.ErrSprintClosed):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "sprint_closed", Message: "sprint is closed"})
	case errors.Is(err, application.ErrStoryNotInSprint):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "story_not_in_sprint", Message: "story is not assigned to the selected sprint"})
	case errors.Is(err, application.ErrStoryAlreadyCompleted):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "story_already_completed", Message: "story is already completed in the selected sprint"})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal_error", Message: "an unexpected error occurred"})
	}
}
