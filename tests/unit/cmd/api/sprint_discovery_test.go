package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	sprintapplication "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	sprintdomain "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
	storyapplication "github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storydomain "github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

const discoveryProjectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
const discoverySprintID = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"

type fakeOpenSprintLister struct {
	calls   int
	id      string
	sprints []sprintdomain.Sprint
	err     error
}

func (l *fakeOpenSprintLister) ListOpenByProject(_ context.Context, projectID string) ([]sprintdomain.Sprint, error) {
	l.calls++
	l.id = projectID
	return l.sprints, l.err
}

type fakeAssignedStoryLister struct {
	calls     int
	projectID string
	sprintID  string
	stories   []storydomain.AssignedSprintStory
	err       error
}

func (l *fakeAssignedStoryLister) ListAssignedToSprint(_ context.Context, projectID, sprintID string) ([]storydomain.AssignedSprintStory, error) {
	l.calls++
	l.projectID, l.sprintID = projectID, sprintID
	return l.stories, l.err
}

func TestSprintDiscoveryRoutesReturnScopedReadOnlyResults(t *testing.T) {
	openLister := &fakeOpenSprintLister{sprints: []sprintdomain.Sprint{{ID: discoverySprintID, ProjectID: discoveryProjectID, SprintGoal: "Deliver"}}}
	assignedLister := &fakeAssignedStoryLister{stories: []storydomain.AssignedSprintStory{{ID: "11111111-1111-4111-8111-111111111111", Title: "Build view", Priority: "alta"}}}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return discoveryProjectID }, api.HTTPDependencies{
		Sprints: &api.SprintDependencies{Repository: &fakeSprintRepository{}, GenerateID: func() string { return discoverySprintID }, OpenLister: openLister},
		Stories: &api.StoryDependencies{Repository: &fakeStoryRepository{}, GenerateID: func() string { return discoverySprintID }, AssignedLister: assignedLister},
	})

	open := serve(handler, http.MethodGet, "/projects/"+discoveryProjectID+"/sprints/open", "")
	if open.Code != http.StatusOK || !strings.Contains(open.Body.String(), `"sprint_goal":"Deliver"`) || !strings.Contains(open.Body.String(), `"is_closed":false`) {
		t.Fatalf("open sprint listing = %d %s", open.Code, open.Body.String())
	}
	assigned := serve(handler, http.MethodGet, "/projects/"+discoveryProjectID+"/sprints/"+discoverySprintID+"/stories", "")
	if assigned.Code != http.StatusOK || !strings.Contains(assigned.Body.String(), `"title":"Build view"`) || !strings.Contains(assigned.Body.String(), discoverySprintID) {
		t.Fatalf("assigned story listing = %d %s", assigned.Code, assigned.Body.String())
	}
	if openLister.calls != 1 || openLister.id != discoveryProjectID || assignedLister.calls != 1 || assignedLister.projectID != discoveryProjectID || assignedLister.sprintID != discoverySprintID {
		t.Fatalf("unexpected read scope: open=%+v assigned=%+v", openLister, assignedLister)
	}
}

func TestSprintDiscoveryRoutesHandleEmptyAndInvalidOrMissingScopes(t *testing.T) {
	for _, test := range []struct {
		name        string
		path        string
		openErr     error
		assignedErr error
		wantStatus  int
		wantCode    string
	}{
		{name: "open list empty", path: "/projects/" + discoveryProjectID + "/sprints/open", wantStatus: http.StatusOK},
		{name: "assigned story list empty", path: "/projects/" + discoveryProjectID + "/sprints/" + discoverySprintID + "/stories", wantStatus: http.StatusOK},
		{name: "invalid project", path: "/projects/not-a-uuid/sprints/open", wantStatus: http.StatusUnprocessableEntity, wantCode: `"error":"validation_failed"`},
		{name: "missing project", path: "/projects/" + discoveryProjectID + "/sprints/open", openErr: sprintapplication.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: `"error":"project_not_found"`},
		{name: "invalid sprint", path: "/projects/" + discoveryProjectID + "/sprints/not-a-uuid/stories", wantStatus: http.StatusUnprocessableEntity, wantCode: `"error":"validation_failed"`},
		{name: "missing sprint", path: "/projects/" + discoveryProjectID + "/sprints/" + discoverySprintID + "/stories", assignedErr: storyapplication.ErrSprintNotFound, wantStatus: http.StatusNotFound, wantCode: `"error":"sprint_not_found"`},
		{name: "missing project in sprint scope", path: "/projects/" + discoveryProjectID + "/sprints/" + discoverySprintID + "/stories", assignedErr: storyapplication.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: `"error":"project_not_found"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			openLister := &fakeOpenSprintLister{err: test.openErr}
			assignedLister := &fakeAssignedStoryLister{err: test.assignedErr}
			handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return discoveryProjectID }, api.HTTPDependencies{
				Sprints: &api.SprintDependencies{Repository: &fakeSprintRepository{}, GenerateID: func() string { return discoverySprintID }, OpenLister: openLister},
				Stories: &api.StoryDependencies{Repository: &fakeStoryRepository{}, GenerateID: func() string { return discoverySprintID }, AssignedLister: assignedLister},
			})
			response := serve(handler, http.MethodGet, test.path, "")
			if response.Code != test.wantStatus || (test.wantCode != "" && !strings.Contains(response.Body.String(), test.wantCode)) {
				t.Fatalf("GET %s = %d %s, want %d containing %s", test.path, response.Code, response.Body.String(), test.wantStatus, test.wantCode)
			}
			if test.name == "open list empty" && !strings.Contains(response.Body.String(), `"sprints":[]`) {
				t.Errorf("empty open sprints = %s, want an empty array", response.Body.String())
			}
			if test.name == "assigned story list empty" && !strings.Contains(response.Body.String(), `"stories":[]`) {
				t.Errorf("empty assigned stories = %s, want an empty array", response.Body.String())
			}
		})
	}
}

func TestOpenSprintDiscoveryDoesNotReplaceClosedHistoryRoute(t *testing.T) {
	history := &fakeSprintHistoryReader{sprints: []sprintdomain.ClosedSprint{{ID: discoverySprintID, ProjectID: discoveryProjectID, SprintGoal: "Closed"}}}
	openLister := &fakeOpenSprintLister{sprints: []sprintdomain.Sprint{{ID: "11111111-1111-4111-8111-111111111111", ProjectID: discoveryProjectID, SprintGoal: "Open"}}}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return discoveryProjectID }, api.HTTPDependencies{
		Sprints: &api.SprintDependencies{Repository: &fakeSprintRepository{}, GenerateID: func() string { return discoverySprintID }, Lister: history, OpenLister: openLister},
	})
	closed := serve(handler, http.MethodGet, "/projects/"+discoveryProjectID+"/sprints", "")
	open := serve(handler, http.MethodGet, "/projects/"+discoveryProjectID+"/sprints/open", "")
	if closed.Code != http.StatusOK || !strings.Contains(closed.Body.String(), `"sprint_goal":"Closed"`) || strings.Contains(closed.Body.String(), `"sprint_goal":"Open"`) {
		t.Fatalf("closed history route changed: %d %s", closed.Code, closed.Body.String())
	}
	if open.Code != http.StatusOK || !strings.Contains(open.Body.String(), `"sprint_goal":"Open"`) || strings.Contains(open.Body.String(), `"sprint_goal":"Closed"`) {
		t.Fatalf("open listing incorrect: %d %s", open.Code, open.Body.String())
	}
}
