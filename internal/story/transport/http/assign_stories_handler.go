package transporthttp

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
)

type assignStoriesRequest struct {
	StoryIDs []string `json:"story_ids"`
}

type AssignStoriesHandler struct {
	useCase *application.AssignStoriesUseCase
}

func NewAssignStoriesHandler(useCase *application.AssignStoriesUseCase) http.Handler {
	return &AssignStoriesHandler{useCase: useCase}
}

func (h *AssignStoriesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed", Message: "only POST is supported"})
		return
	}
	var input assignStoriesRequest
	if _, err := decodeStoryObject(r.Body, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_request", Message: "request body must be a single valid JSON object with allowed fields"})
		return
	}
	projectID, err := uuid.Parse(r.PathValue("project_id"))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "validation_failed", Message: "one or more fields are invalid", Fields: map[string]string{"project_id": "must be a valid UUID"}})
		return
	}
	sprintID, err := uuid.Parse(r.PathValue("sprint_id"))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "validation_failed", Message: "one or more fields are invalid", Fields: map[string]string{"sprint_id": "must be a valid UUID"}})
		return
	}
	for i, id := range input.StoryIDs {
		storyID, err := uuid.Parse(id)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "validation_failed", Message: "one or more fields are invalid", Fields: map[string]string{"story_ids": "must contain valid UUIDs"}})
			return
		}
		input.StoryIDs[i] = storyID.String()
	}
	err = h.useCase.Execute(r.Context(), application.AssignStoriesCommand{ProjectID: projectID.String(), SprintID: sprintID.String(), StoryIDs: input.StoryIDs})
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, map[string]any{"sprint_id": sprintID.String(), "story_ids": input.StoryIDs})
	case errors.Is(err, application.ErrSprintNotFound), errors.Is(err, application.ErrStoryNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "resource_not_found", Message: "sprint or story not found"})
	case errors.Is(err, application.ErrSprintClosed), errors.Is(err, application.ErrProjectMismatch), errors.Is(err, application.ErrStoryAlreadyAssigned):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "assignment_conflict", Message: "the selected stories cannot be assigned to this sprint"})
	case errors.Is(err, application.ErrEmptyStorySelection), errors.Is(err, application.ErrDuplicateStoryID):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: "validation_failed", Message: "one or more fields are invalid", Fields: map[string]string{"story_ids": err.Error()}})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal_error", Message: "an unexpected error occurred"})
	}
}
