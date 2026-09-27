package transporthttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

type CreateSprintHandler struct {
	useCase *application.CreateSprintUseCase
}

func NewCreateSprintHandler(useCase *application.CreateSprintUseCase) http.Handler {
	return &CreateSprintHandler{useCase: useCase}
}

type createSprintRequest struct {
	SprintGoal string `json:"sprint_goal"`
}

type sprintResponse struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	SprintGoal string `json:"sprint_goal"`
}

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (h *CreateSprintHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed", Message: "only POST is supported"})
		return
	}
	input, err := decodeCreateSprintRequest(request.Body)
	if err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Error: "invalid_request", Message: "request body must be a single valid JSON object with allowed fields",
		})
		return
	}
	projectID, err := uuid.Parse(request.PathValue("project_id"))
	if err != nil {
		writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
			Error: "validation_failed", Message: "one or more fields are invalid",
			Fields: map[string]string{"project_id": "must be a valid UUID"},
		})
		return
	}
	sprint, err := h.useCase.Execute(request.Context(), application.CreateSprintCommand{
		ProjectID: projectID.String(), SprintGoal: input.SprintGoal,
	})
	if err != nil {
		var validation *domain.ValidationError
		switch {
		case errors.As(err, &validation):
			writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
				Error: "validation_failed", Message: "one or more fields are invalid", Fields: validation.Fields,
			})
		case errors.Is(err, application.ErrProjectNotFound):
			writeJSON(response, http.StatusNotFound, errorResponse{Error: "project_not_found", Message: "project not found"})
		default:
			writeJSON(response, http.StatusInternalServerError, errorResponse{Error: "internal_error", Message: "an unexpected error occurred"})
		}
		return
	}
	writeJSON(response, http.StatusCreated, sprintResponse{ID: sprint.ID, ProjectID: sprint.ProjectID, SprintGoal: sprint.SprintGoal})
}

func decodeCreateSprintRequest(body io.Reader) (createSprintRequest, error) {
	decoder := json.NewDecoder(body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return createSprintRequest{}, err
	}
	if len(raw) == 0 || raw[0] != '{' {
		return createSprintRequest{}, errors.New("request must be an object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return createSprintRequest{}, errors.New("request must contain one object")
	}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	var input createSprintRequest
	if err := strict.Decode(&input); err != nil {
		return createSprintRequest{}, err
	}
	return input, nil
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}
