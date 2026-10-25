// Package transporthttp exposes defect registration over HTTP.
package transporthttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

const (
	validationMessage  = "one or more fields are invalid"
	invalidUUIDMessage = "must be a valid UUID"
)

// RegisterDefectHandler handles the registration of a defect detected during development.
type RegisterDefectHandler struct {
	useCase *application.RegisterDefectUseCase
}

// NewRegisterDefectHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewRegisterDefectHandler(useCase *application.RegisterDefectUseCase) http.Handler {
	return &RegisterDefectHandler{useCase: useCase}
}

type registerDefectRequest struct {
	StoryID     string `json:"story_id"`
	SprintID    string `json:"sprint_id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
}

// defectResponse is the flat 201 body: exactly the seven contract keys. created_at and seq
// stay internal to storage and are deliberately not exposed.
type defectResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	StoryID     string `json:"story_id"`
	SprintID    string `json:"sprint_id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
}

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (h *RegisterDefectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only POST is supported",
		})
		return
	}

	var input registerDefectRequest
	if err := decodeDefectObject(request.Body, &input); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Error: "invalid_request", Message: "request body must be a single valid JSON object with allowed fields",
		})
		return
	}

	projectID, err := uuid.Parse(request.PathValue("project_id"))
	if err != nil {
		writeFieldError(response, "project_id", invalidUUIDMessage)
		return
	}

	defect, err := h.useCase.Execute(request.Context(), application.RegisterDefectCommand{
		ProjectID:   projectID.String(),
		StoryID:     input.StoryID,
		SprintID:    input.SprintID,
		Description: input.Description,
		Severity:    input.Severity,
		Status:      input.Status,
	})
	if err != nil {
		writeUseCaseError(response, err)
		return
	}

	writeJSON(response, http.StatusCreated, newDefectResponse(defect))
}

// writeUseCaseError maps an application error to its public response; unknown errors are
// reported generically so internal details are never leaked.
func writeUseCaseError(response http.ResponseWriter, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
			Error: "validation_failed", Message: validationMessage, Fields: validation.Fields,
		})
	case errors.Is(err, application.ErrProjectNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "project_not_found", Message: "project not found"})
	case errors.Is(err, application.ErrStoryNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "story_not_found", Message: "story not found"})
	case errors.Is(err, application.ErrSprintNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "sprint_not_found", Message: "sprint not found"})
	default:
		writeJSON(response, http.StatusInternalServerError, errorResponse{
			Error: "internal_error", Message: "an unexpected error occurred",
		})
	}
}

// newDefectResponse projects the stored record to the public response.
func newDefectResponse(defect domain.Defect) defectResponse {
	return defectResponse{
		ID:          defect.ID,
		ProjectID:   defect.ProjectID,
		StoryID:     defect.StoryID,
		SprintID:    defect.SprintID,
		Description: defect.Description,
		Severity:    defect.Severity,
		Status:      defect.Status,
	}
}

// decodeDefectObject strictly decodes a single JSON object into target: the first value must
// be an object, nothing may follow it, and unknown fields or wrong types are rejected.
func decodeDefectObject(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	if len(raw) == 0 || raw[0] != '{' {
		return errors.New("request must be an object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request must contain one object")
	}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	return strict.Decode(target)
}

func writeFieldError(response http.ResponseWriter, field, message string) {
	writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
		Error: "validation_failed", Message: validationMessage, Fields: map[string]string{field: message},
	})
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}
