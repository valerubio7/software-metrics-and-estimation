// Package transporthttp exposes task creation over HTTP.
package transporthttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

// CreateTasksHandler handles batch task creation for a story within a sprint.
type CreateTasksHandler struct {
	useCase *application.CreateTasksUseCase
}

// NewCreateTasksHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewCreateTasksHandler(useCase *application.CreateTasksUseCase) http.Handler {
	return &CreateTasksHandler{useCase: useCase}
}

type createTasksRequest struct {
	Tasks []taskRequest `json:"tasks"`
}

type taskRequest struct {
	Title          string   `json:"title"`
	EstimatedHours *float64 `json:"estimated_hours"`
}

type taskResponse struct {
	ID             string   `json:"id"`
	ProjectID      string   `json:"project_id"`
	SprintID       string   `json:"sprint_id"`
	StoryID        string   `json:"story_id"`
	Title          string   `json:"title"`
	EstimatedHours *float64 `json:"estimated_hours"`
}

type createTasksResponse struct {
	Tasks []taskResponse `json:"tasks"`
}

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (h *CreateTasksHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only POST is supported",
		})
		return
	}

	var input createTasksRequest
	if _, err := decodeTaskObject(request.Body, &input); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Error: "invalid_request", Message: "request body must be a single valid JSON object with allowed fields",
		})
		return
	}

	// A batch with no tasks is a structural request problem, not per-element content
	// validation, so it is rejected here before route IDs or the use case are evaluated.
	if len(input.Tasks) == 0 {
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
	sprintID, err := uuid.Parse(request.PathValue("sprint_id"))
	if err != nil {
		writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
			Error: "validation_failed", Message: "one or more fields are invalid",
			Fields: map[string]string{"sprint_id": "must be a valid UUID"},
		})
		return
	}
	storyID, err := uuid.Parse(request.PathValue("story_id"))
	if err != nil {
		writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
			Error: "validation_failed", Message: "one or more fields are invalid",
			Fields: map[string]string{"story_id": "must be a valid UUID"},
		})
		return
	}

	command := application.CreateTasksCommand{
		ProjectID: projectID.String(), SprintID: sprintID.String(), StoryID: storyID.String(),
		Tasks: make([]application.TaskInput, len(input.Tasks)),
	}
	for i, task := range input.Tasks {
		command.Tasks[i] = application.TaskInput{Title: task.Title, EstimatedHours: task.EstimatedHours}
	}

	tasks, err := h.useCase.Execute(request.Context(), command)
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
		case errors.Is(err, application.ErrSprintNotFound):
			writeJSON(response, http.StatusNotFound, errorResponse{
				Error: "sprint_not_found", Message: "sprint not found",
			})
		case errors.Is(err, application.ErrStoryNotFound):
			writeJSON(response, http.StatusNotFound, errorResponse{
				Error: "story_not_found", Message: "story not found",
			})
		case errors.Is(err, application.ErrStoryNotInSprint):
			writeJSON(response, http.StatusConflict, errorResponse{
				Error: "story_not_in_sprint", Message: "story is not assigned to the selected sprint",
			})
		default:
			writeJSON(response, http.StatusInternalServerError, errorResponse{
				Error: "internal_error", Message: "an unexpected error occurred",
			})
		}
		return
	}

	writeJSON(response, http.StatusCreated, newCreateTasksResponse(tasks))
}

// newCreateTasksResponse projects created tasks to the public response, in request order.
func newCreateTasksResponse(tasks []domain.Task) createTasksResponse {
	out := createTasksResponse{Tasks: make([]taskResponse, len(tasks))}
	for i, task := range tasks {
		out.Tasks[i] = taskResponse{
			ID: task.ID, ProjectID: task.ProjectID, SprintID: task.SprintID, StoryID: task.StoryID,
			Title: task.Title, EstimatedHours: task.EstimatedHours,
		}
	}
	return out
}

// decodeTaskObject strictly decodes a single JSON object into target and returns its raw
// keys, replicating the technique of internal/story/transport/http/handler.go's
// decodeStoryObject: a single object, DisallowUnknownFields (which also applies to the
// nested objects of tasks), and rejecting null elements inside the tasks array.
func decodeTaskObject(body io.Reader, target any) (map[string]json.RawMessage, error) {
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
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if tasks := fields["tasks"]; len(tasks) > 0 && tasks[0] == '[' {
		var elements []json.RawMessage
		if err := json.Unmarshal(tasks, &elements); err != nil {
			return nil, err
		}
		for _, element := range elements {
			if bytes.Equal(element, []byte("null")) {
				return nil, errors.New("task must be an object")
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
