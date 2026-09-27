package transporthttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

// CreateStoryHandler handles creation for a project identified by the route.
type CreateStoryHandler struct {
	useCase *application.CreateStoryUseCase
}

// NewCreateStoryHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewCreateStoryHandler(useCase *application.CreateStoryUseCase) http.Handler {
	return &CreateStoryHandler{useCase: useCase}
}

type createStoryRequest struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type storyResponse struct {
	ID                 string   `json:"id"`
	ProjectID          string   `json:"project_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	StoryPoints        *int     `json:"story_points"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EstimatedHours     *float64 `json:"estimated_hours"`
}

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (h *CreateStoryHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only POST is supported",
		})
		return
	}

	var input createStoryRequest
	if _, err := decodeStoryObject(request.Body, &input); err != nil {
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

	story, err := h.useCase.Execute(request.Context(), application.CreateStoryCommand{
		ProjectID: projectID.String(), Title: input.Title, Description: input.Description,
		Priority: input.Priority, AcceptanceCriteria: input.AcceptanceCriteria,
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

	writeJSON(response, http.StatusCreated, newStoryResponse(story))
}

// newStoryResponse projects a story to the public response shared by every story operation.
func newStoryResponse(story domain.Story) storyResponse {
	return storyResponse{
		ID: story.ID, ProjectID: story.ProjectID, Title: story.Title,
		Description: story.Description, Priority: story.Priority, Status: story.Status,
		StoryPoints: story.StoryPoints, AcceptanceCriteria: story.AcceptanceCriteria,
		EstimatedHours: story.EstimatedHours,
	}
}

// decodeStoryObject strictly decodes a single JSON object into target and returns its raw keys,
// so callers can tell an absent key apart from an explicit null.
func decodeStoryObject(body io.Reader, target any) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || raw[0] != '{' {
		return nil, errors.New("request must be an object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("request must contain one object")
	}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(target); err != nil {
		return nil, err
	}
	// JSON null inside a string array otherwise decodes as an empty string.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if criteria := fields["acceptance_criteria"]; len(criteria) > 0 && criteria[0] == '[' {
		var elements []json.RawMessage
		if err := json.Unmarshal(criteria, &elements); err != nil {
			return nil, err
		}
		for _, element := range elements {
			if bytes.Equal(element, []byte("null")) {
				return nil, errors.New("criterion must be a string")
			}
		}
	}
	return fields, nil
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}
