package transporthttp

import (
	"errors"
	"net/http"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

// ListSprintsHandler serves the closed sprint history of the project identified by the route.
type ListSprintsHandler struct {
	useCase *application.ListSprintsUseCase
}

// NewListSprintsHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewListSprintsHandler(useCase *application.ListSprintsUseCase) http.Handler {
	return &ListSprintsHandler{useCase: useCase}
}

// sprintHistoryResponse is the container of the history, so paging metadata can be added later.
type sprintHistoryResponse struct {
	ProjectID string                 `json:"project_id"`
	Sprints   []closedSprintResponse `json:"sprints"`
}

type closedSprintResponse struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	SprintGoal string                 `json:"sprint_goal"`
	IsClosed   bool                   `json:"is_closed"`
	Stories    []plannedStoryResponse `json:"stories"`
}

// plannedStoryResponse keeps story_points and completed_at without omitempty: a missing value is an explicit null.
type plannedStoryResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Priority    string     `json:"priority"`
	StoryPoints *int       `json:"story_points"`
	CompletedAt *time.Time `json:"completed_at"`
	Completed   bool       `json:"completed"`
}

// utcCopy returns the instant in UTC without mutating the domain value; nil stays nil (explicit null).
func utcCopy(instant *time.Time) *time.Time {
	if instant == nil {
		return nil
	}
	utc := instant.UTC()
	return &utc
}

func (h *ListSprintsHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// A GET route also serves HEAD, so both methods must reach the handler.
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only GET is supported",
		})
		return
	}

	history, err := h.useCase.Execute(request.Context(), application.ListSprintsQuery{
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
			writeJSON(response, http.StatusNotFound, errorResponse{Error: "project_not_found", Message: "project not found"})
		default:
			writeJSON(response, http.StatusInternalServerError, errorResponse{
				Error: "internal_error", Message: "an unexpected error occurred",
			})
		}
		return
	}

	// Non-nil slices serialize as [] even when there are no sprints or stories.
	sprints := make([]closedSprintResponse, 0, len(history.Sprints))
	for _, sprint := range history.Sprints {
		stories := make([]plannedStoryResponse, 0, len(sprint.Stories))
		for _, story := range sprint.Stories {
			stories = append(stories, plannedStoryResponse{
				ID: story.ID, Title: story.Title, Priority: story.Priority,
				StoryPoints: story.StoryPoints, CompletedAt: utcCopy(story.CompletedAt), Completed: story.Completed(),
			})
		}
		sprints = append(sprints, closedSprintResponse{
			ID: sprint.ID, ProjectID: sprint.ProjectID, SprintGoal: sprint.SprintGoal, IsClosed: true, Stories: stories,
		})
	}
	writeJSON(response, http.StatusOK, sprintHistoryResponse{ProjectID: history.ProjectID, Sprints: sprints})
}
