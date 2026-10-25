// Package transporthttp exposes effort registration over HTTP.
package transporthttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/domain"
)

const (
	workDateLayout = "2006-01-02"

	validationMessage  = "one or more fields are invalid"
	invalidUUIDMessage = "must be a valid UUID"
)

// RegisterEffortHandler handles the registration of effort over a story or one of its tasks.
type RegisterEffortHandler struct {
	useCase *application.RegisterEffortUseCase
}

// NewRegisterEffortHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewRegisterEffortHandler(useCase *application.RegisterEffortUseCase) http.Handler {
	return &RegisterEffortHandler{useCase: useCase}
}

type registerEffortRequest struct {
	MemberID    string   `json:"member_id"`
	TaskID      *string  `json:"task_id"`
	WorkDate    string   `json:"work_date"`
	Description string   `json:"description"`
	Hours       *float64 `json:"hours"`
}

// effortRecordResponse is the flat 201 body. task_id is always present (null for effort over
// the story itself) and created_at is deliberately not exposed.
type effortRecordResponse struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	StoryID     string  `json:"story_id"`
	TaskID      *string `json:"task_id"`
	MemberID    string  `json:"member_id"`
	WorkDate    string  `json:"work_date"`
	Description string  `json:"description"`
	Hours       float64 `json:"hours"`
}

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (h *RegisterEffortHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only POST is supported",
		})
		return
	}

	var input registerEffortRequest
	if err := decodeEffortObject(request.Body, &input); err != nil {
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
	storyID, err := uuid.Parse(request.PathValue("story_id"))
	if err != nil {
		writeFieldError(response, "story_id", invalidUUIDMessage)
		return
	}

	record, err := h.useCase.Execute(request.Context(), application.RegisterEffortCommand{
		ProjectID:   projectID.String(),
		StoryID:     storyID.String(),
		MemberID:    input.MemberID,
		TaskID:      input.TaskID,
		WorkDate:    input.WorkDate,
		Description: input.Description,
		Hours:       input.Hours,
	})
	if err != nil {
		writeUseCaseError(response, err)
		return
	}

	writeJSON(response, http.StatusCreated, newEffortRecordResponse(record))
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
	case errors.Is(err, application.ErrMemberNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "member_not_found", Message: "member not found"})
	case errors.Is(err, application.ErrTaskNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{Error: "task_not_found", Message: "task not found"})
	default:
		writeJSON(response, http.StatusInternalServerError, errorResponse{
			Error: "internal_error", Message: "an unexpected error occurred",
		})
	}
}

// newEffortRecordResponse projects the stored record to the public response.
func newEffortRecordResponse(record domain.EffortRecord) effortRecordResponse {
	return effortRecordResponse{
		ID:          record.ID,
		ProjectID:   record.ProjectID,
		StoryID:     record.StoryID,
		TaskID:      record.TaskID,
		MemberID:    record.MemberID,
		WorkDate:    record.WorkDate.Format(workDateLayout),
		Description: record.Description,
		Hours:       record.Hours,
	}
}

// decodeEffortObject strictly decodes a single JSON object into target: the first value must
// be an object, nothing may follow it, and unknown fields or wrong types are rejected.
func decodeEffortObject(body io.Reader, target any) error {
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
