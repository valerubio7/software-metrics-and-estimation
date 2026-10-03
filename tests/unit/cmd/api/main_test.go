package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
	memberdomain "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
	sprintdomain "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
	storyapplication "github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storydomain "github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
	taskdomain "github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

type fakeProjectRepository struct {
	projects    []domain.Project
	updateCalls int
	updateErr   error
}

func (r *fakeProjectRepository) Create(_ context.Context, project domain.Project) error {
	r.projects = append(r.projects, project)
	return nil
}

func (r *fakeProjectRepository) Update(_ context.Context, project domain.Project) error {
	r.updateCalls++
	if r.updateErr != nil {
		return r.updateErr
	}
	r.projects = append(r.projects, project)
	return nil
}

type fakeStoryRepository struct {
	stories []storydomain.Story
}

type fakeSprintRepository struct {
	sprints []sprintdomain.Sprint
}

type fakeMemberRepository struct{ members []memberdomain.Member }

func (r *fakeMemberRepository) Register(_ context.Context, _ string, members []memberdomain.Member) error {
	r.members = append(r.members, members...)
	return nil
}

func (r *fakeSprintRepository) Create(_ context.Context, sprint sprintdomain.Sprint) error {
	r.sprints = append(r.sprints, sprint)
	return nil
}

func TestSprintCompositionIsOptionalAndPreservesExistingRoutes(t *testing.T) {
	projectID := "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	projects := &fakeProjectRepository{}
	stories := &fakeStoryRepository{}
	sprints := &fakeSprintRepository{}
	projectGenerator := func() string { return projectID }
	withoutSprint := api.NewHTTPHandlerWithDependencies(projects, projectGenerator, api.HTTPDependencies{
		Stories: &api.StoryDependencies{Repository: stories, GenerateID: projectGenerator},
	})
	response := httptest.NewRecorder()
	withoutSprint.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/sprints", strings.NewReader(`{"sprint_goal":"Deliver"}`)))
	if response.Code != http.StatusNotFound {
		t.Fatalf("sprint route without dependencies = %d", response.Code)
	}

	handler := api.NewHTTPHandlerWithDependencies(projects, projectGenerator, api.HTTPDependencies{
		Stories: &api.StoryDependencies{Repository: stories, GenerateID: projectGenerator},
		Sprints: &api.SprintDependencies{Repository: sprints, GenerateID: projectGenerator},
	})
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/sprints", strings.NewReader(`{"sprint_goal":"Deliver"}`)))
	if created.Code != http.StatusCreated || len(sprints.sprints) != 1 {
		t.Fatalf("sprint status=%d writes=%d body=%s", created.Code, len(sprints.sprints), created.Body.String())
	}
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/projects/"+projectID+"/sprints", strings.NewReader(`{"sprint_goal":"Must not write"}`)))
		if response.Code != http.StatusMethodNotAllowed || len(sprints.sprints) != 1 {
			t.Errorf("%s status=%d writes=%d", method, response.Code, len(sprints.sprints))
		}
	}
	story := httptest.NewRecorder()
	handler.ServeHTTP(story, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/stories", strings.NewReader(`{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Listo"]}`)))
	project := httptest.NewRecorder()
	handler.ServeHTTP(project, httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`)))
	if story.Code != http.StatusCreated || len(stories.stories) != 1 || project.Code != http.StatusCreated || len(projects.projects) != 1 {
		t.Fatalf("preserved composition: story=%d/%d project=%d/%d", story.Code, len(stories.stories), project.Code, len(projects.projects))
	}
}

type readableProjectRepository struct {
	*fakeProjectRepository
	project domain.Project
	reads   int
}

func (r *readableProjectRepository) GetByID(_ context.Context, id string) (domain.Project, error) {
	r.reads++
	if id != r.project.ID {
		return domain.Project{}, application.ErrProjectNotFound
	}
	return r.project, nil
}

func TestProjectStatusRouteIsComposedOnlyForReadableRepository(t *testing.T) {
	const id = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	today := time.Now().UTC()
	readable := &readableProjectRepository{fakeProjectRepository: &fakeProjectRepository{}, project: domain.Project{
		ID: id, Name: "Metrics", StartDate: today.AddDate(0, 0, -1), PlannedFinishDate: today.AddDate(0, 0, 1),
	}}
	handler := api.NewHTTPHandler(readable, func() string { return "generated" })
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/projects/"+id, nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"active"`) || readable.reads != 1 {
		t.Fatalf("GET status=%d reads=%d body=%s", response.Code, readable.reads, response.Body.String())
	}

	legacy := api.NewHTTPHandler(&fakeProjectRepository{}, func() string { return "generated" })
	missingRoute := httptest.NewRecorder()
	legacy.ServeHTTP(missingRoute, httptest.NewRequest(http.MethodGet, "/projects/"+id, nil))
	if missingRoute.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET without read dependency = %d, want 405", missingRoute.Code)
	}
}

func TestMemberRouteRequiresExplicitDependenciesAndCleanVersionSix(t *testing.T) {
	id := "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	for _, tc := range []struct {
		name    string
		version int
		dirty   bool
		lookup  error
		enabled bool
	}{
		{"clean v5", 5, false, nil, false}, {"clean v6", 6, false, nil, true}, {"dirty v6", 6, true, nil, false}, {"lookup error", 6, false, errors.New("query failed"), false},
		{"missing migration table", 0, false, errors.New(`relation "schema_migrations" does not exist`), false},
		{"no migration version row", 0, false, errors.New("no rows in result set"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(tc.version, tc.dirty, tc.lookup)
			if readiness.Members != tc.enabled {
				t.Fatalf("members readiness=%t", readiness.Members)
			}
			repo := &fakeMemberRepository{}
			deps := api.HTTPDependencies{}
			if tc.enabled {
				deps.Members = &api.MemberDependencies{Repository: repo, GenerateID: func() string { return id }}
			}
			h := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return id }, deps)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/projects/"+id+"/members", strings.NewReader(`{"members":[{"full_name":"Ada"}]}`)))
			want := http.StatusNotFound
			if tc.enabled {
				want = http.StatusCreated
			}
			if w.Code != want || (tc.enabled && len(repo.members) != 1) {
				t.Fatalf("route status=%d writes=%d want=%d", w.Code, len(repo.members), want)
			}
		})
	}
}

type fakeAssignmentRepository struct{ calls int }

func (r *fakeAssignmentRepository) AssignStories(context.Context, string, []string) error {
	r.calls++
	return errors.New("unscoped assignment must not be called")
}
func (r *fakeAssignmentRepository) AssignStoriesForProject(context.Context, string, string, []string) error {
	r.calls++
	return nil
}

func TestAssignmentRequiresCleanVersionEightAndExplicitDependency(t *testing.T) {
	const id = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	for _, tc := range []struct {
		name          string
		version       int
		dirty         bool
		lookup        error
		nilDependency bool
		enabled       bool
	}{
		{name: "v6", version: 6},
		{name: "v7", version: 7},
		{name: "v8", version: 8, enabled: true},
		{name: "future", version: 12, enabled: true},
		{name: "nil dependency", version: 8, nilDependency: true},
		{name: "dirty", version: 8, dirty: true},
		{name: "lookup error", version: 8, lookup: errors.New("query failed")},
		{name: "missing migration table", lookup: errors.New("missing schema_migrations")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(tc.version, tc.dirty, tc.lookup)
			wantReady := tc.version >= 8 && !tc.dirty && tc.lookup == nil
			if readiness.Assignment != wantReady {
				t.Fatalf("readiness = %+v", readiness)
			}
			repo := &fakeAssignmentRepository{}
			deps := api.HTTPDependencies{
				Stories: &api.StoryDependencies{Repository: &fakeStoryRepository{}, GenerateID: func() string { return id }},
				Sprints: &api.SprintDependencies{Repository: &fakeSprintRepository{}, GenerateID: func() string { return id }},
			}
			// Use the actual startup resolver, never infer readiness from repository type.
			if readiness.Assignment && !tc.nilDependency {
				deps.Stories.Assigner = repo
			}
			h := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return id }, deps)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/projects/"+id+"/sprints/"+id+"/stories", strings.NewReader(`{"story_ids":["`+id+`"]}`)))
			wantStatus, wantCalls := http.StatusNotFound, 0
			if tc.enabled {
				wantStatus, wantCalls = http.StatusCreated, 1
			}
			if w.Code != wantStatus || repo.calls != wantCalls {
				t.Fatalf("status/writes = %d/%d want %d/%d", w.Code, repo.calls, wantStatus, wantCalls)
			}
		})
	}
}

func TestMigrationReadinessSelectsRoutesIndependently(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		version int
		dirty   bool
		lookup  error
		stories bool
		sprints bool
		members bool
		tasks   bool
	}{
		{name: "clean v1", version: 1},
		{name: "clean v2", version: 2, stories: true},
		{name: "clean v3", version: 3, stories: true},
		{name: "clean v4", version: 4, stories: true},
		{name: "clean v5", version: 5, stories: true, sprints: true},
		{name: "clean v6", version: 6, stories: true, sprints: true, members: true},
		{name: "clean v8", version: 8, stories: true, sprints: true, members: true},
		{name: "clean v9", version: 9, stories: true, sprints: true, members: true, tasks: true},
		{name: "clean future", version: 12, stories: true, sprints: true, members: true, tasks: true},
		{name: "dirty v3", version: 3, dirty: true},
		{name: "dirty v5", version: 5, dirty: true},
		{name: "dirty v6", version: 6, dirty: true},
		{name: "dirty v9", version: 9, dirty: true},
		{name: "lookup error", lookup: context.DeadlineExceeded},
		{name: "missing migration table", lookup: errors.New(`relation "schema_migrations" does not exist`)},
		{name: "no migration version row", lookup: errors.New("no rows in result set")},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(scenario.version, scenario.dirty, scenario.lookup)
			if readiness.Stories != scenario.stories || readiness.Sprints != scenario.sprints || readiness.Members != scenario.members || readiness.Tasks != scenario.tasks || !readiness.Projects {
				t.Errorf("readiness = %+v", readiness)
			}

			id := "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
			generateID := func() string { return id }
			projects := &fakeProjectRepository{}
			stories := &fakeStoryRepository{}
			sprints := &fakeSprintRepository{}
			members := &fakeMemberRepository{}
			deps := api.HTTPDependencies{}
			if readiness.Stories {
				deps.Stories = &api.StoryDependencies{Repository: stories, GenerateID: generateID}
			}
			if readiness.Sprints {
				deps.Sprints = &api.SprintDependencies{Repository: sprints, GenerateID: generateID}
			}
			if readiness.Members {
				deps.Members = &api.MemberDependencies{Repository: members, GenerateID: generateID}
			}
			handler := api.NewHTTPHandlerWithDependencies(projects, generateID, deps)
			for _, route := range []struct {
				name    string
				path    string
				body    string
				enabled bool
				writes  func() int
			}{
				{"projects", "/projects", `{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`, true, func() int { return len(projects.projects) }},
				{"stories", "/projects/" + id + "/stories", `{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Listo"]}`, scenario.stories, func() int { return len(stories.stories) }},
				{"sprints", "/projects/" + id + "/sprints", `{"sprint_goal":"Deliver"}`, scenario.sprints, func() int { return len(sprints.sprints) }},
				{"members", "/projects/" + id + "/members", `{"members":[{"full_name":"Ada"}]}`, scenario.members, func() int { return len(members.members) }},
			} {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, route.path, strings.NewReader(route.body)))
				wantStatus, wantWrites := http.StatusNotFound, 0
				if route.enabled {
					wantStatus, wantWrites = http.StatusCreated, 1
				}
				if response.Code != wantStatus || route.writes() != wantWrites {
					t.Errorf("%s status=%d writes=%d; want status=%d writes=%d; body=%s", route.name, response.Code, route.writes(), wantStatus, wantWrites, response.Body.String())
				}
			}
		})
	}
}

type fakeTaskRepository struct{ calls int }

func (r *fakeTaskRepository) CreateForSprintStory(context.Context, string, string, string, []taskdomain.Task) error {
	r.calls++
	return nil
}

func TestTaskRouteRequiresCleanVersionNineAndExplicitDependency(t *testing.T) {
	const id = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	for _, tc := range []struct {
		name          string
		version       int
		dirty         bool
		lookup        error
		nilDependency bool
		enabled       bool
	}{
		{name: "v7", version: 7},
		{name: "v8", version: 8},
		{name: "v9", version: 9, enabled: true},
		{name: "future 12", version: 12, enabled: true},
		{name: "nil dependency", version: 9, nilDependency: true},
		{name: "dirty v9", version: 9, dirty: true},
		{name: "lookup error", version: 9, lookup: errors.New("query failed")},
		{name: "missing migration table", lookup: errors.New("missing schema_migrations")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(tc.version, tc.dirty, tc.lookup)
			wantReady := tc.version >= 9 && !tc.dirty && tc.lookup == nil
			if readiness.Tasks != wantReady {
				t.Fatalf("readiness = %+v", readiness)
			}
			repo := &fakeTaskRepository{}
			deps := api.HTTPDependencies{}
			if readiness.Tasks && !tc.nilDependency {
				deps.Tasks = &api.TaskDependencies{Repository: repo, GenerateID: func() string { return id }}
			}
			h := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return id }, deps)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/projects/"+id+"/sprints/"+id+"/stories/"+id+"/tasks", strings.NewReader(`{"tasks":[{"title":"A"}]}`)))
			wantStatus, wantCalls := http.StatusNotFound, 0
			if tc.enabled {
				wantStatus, wantCalls = http.StatusCreated, 1
			}
			if w.Code != wantStatus || repo.calls != wantCalls {
				t.Fatalf("status/writes = %d/%d want %d/%d body=%s", w.Code, repo.calls, wantStatus, wantCalls, w.Body.String())
			}
		})
	}
}

func TestTaskRouteIsIndependentFromStories(t *testing.T) {
	const id = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	repo := &fakeTaskRepository{}
	h := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return id }, api.HTTPDependencies{
		Tasks: &api.TaskDependencies{Repository: repo, GenerateID: func() string { return id }},
	})

	taskResponse := httptest.NewRecorder()
	h.ServeHTTP(taskResponse, httptest.NewRequest(http.MethodPost, "/projects/"+id+"/sprints/"+id+"/stories/"+id+"/tasks", strings.NewReader(`{"tasks":[{"title":"A"}]}`)))
	if taskResponse.Code != http.StatusCreated || repo.calls != 1 {
		t.Fatalf("task route status/calls = %d/%d, want 201/1 even without Stories dependency: %s", taskResponse.Code, repo.calls, taskResponse.Body.String())
	}

	storyResponse := httptest.NewRecorder()
	h.ServeHTTP(storyResponse, httptest.NewRequest(http.MethodPost, "/projects/"+id+"/stories", strings.NewReader(`{"title":"T","description":"D","priority":"media","acceptance_criteria":["ok"]}`)))
	if storyResponse.Code != http.StatusNotFound {
		t.Fatalf("story route status = %d, want 404 (Stories dependency absent)", storyResponse.Code)
	}
}

func (r *fakeStoryRepository) Create(_ context.Context, story storydomain.Story) error {
	r.stories = append(r.stories, story)
	return nil
}

type fakeStoryUpdater struct {
	updates []storydomain.Story
}

func (u *fakeStoryUpdater) Update(_ context.Context, story storydomain.Story) (storydomain.Story, error) {
	u.updates = append(u.updates, story)
	return story, nil
}

const (
	updateProjectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	updateStoryID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	updateBody      = `{"title":"Nuevo título","description":"Otra descripción","priority":"alta","status":"en_progreso","acceptance_criteria":["Primero","Segundo"],"estimated_hours":2.5}`
)

func newStoryHandler(stories *fakeStoryRepository, updater *fakeStoryUpdater) http.Handler {
	dependencies := api.StoryDependencies{Repository: stories, GenerateID: func() string { return updateStoryID }}
	if updater != nil {
		dependencies.Updater = updater
	}
	return api.NewHTTPHandler(&fakeProjectRepository{}, func() string { return updateProjectID }, dependencies)
}

// fakeStoryLister records the reads it receives. It exposes reading only, like the port it stands for.
type fakeStoryLister struct {
	calls   int
	lastID  string
	stories []storydomain.Story
	err     error
}

func (l *fakeStoryLister) ListByProject(_ context.Context, projectID string) ([]storydomain.Story, error) {
	l.calls++
	l.lastID = projectID
	return l.stories, l.err
}

// newBacklogHandler composes creation, update and the backlog query, like schema version 4.
func newBacklogHandler(stories *fakeStoryRepository, updater *fakeStoryUpdater, lister *fakeStoryLister) http.Handler {
	return api.NewHTTPHandler(&fakeProjectRepository{}, func() string { return updateProjectID }, api.StoryDependencies{
		Repository: stories,
		GenerateID: func() string { return updateStoryID },
		Updater:    updater,
		Lister:     lister,
	})
}

func backlogStories() []storydomain.Story {
	return []storydomain.Story{
		{ID: "11111111-1111-4111-8111-111111111111", ProjectID: updateProjectID, Title: "Primera", Description: "d", Priority: "baja", Status: "pendiente", AcceptanceCriteria: []string{"a"}},
		{ID: "22222222-2222-4222-8222-222222222222", ProjectID: updateProjectID, Title: "Segunda", Description: "d", Priority: "alta", Status: "en_progreso", AcceptanceCriteria: []string{"a"}},
		{ID: "33333333-3333-4333-8333-333333333333", ProjectID: updateProjectID, Title: "Tercera", Description: "d", Priority: "media", Status: "completada", AcceptanceCriteria: []string{"a"}},
	}
}

func TestBacklogCompositionServesTheOrderedContainerOnCollectionGet(t *testing.T) {
	lister := &fakeStoryLister{stories: backlogStories()}
	handler := newBacklogHandler(&fakeStoryRepository{}, &fakeStoryUpdater{}, lister)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/projects/"+strings.ToUpper(updateProjectID)+"/stories", nil))
	if response.Code != http.StatusOK || lister.calls != 1 {
		t.Fatalf("GET status = %d, reads = %d; body = %s", response.Code, lister.calls, response.Body.String())
	}
	var body struct {
		ProjectID string `json:"project_id"`
		Stories   []struct {
			Title string `json:"title"`
		} `json:"stories"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, story := range body.Stories {
		titles = append(titles, story.Title)
	}
	if body.ProjectID != updateProjectID || lister.lastID != updateProjectID || strings.Join(titles, ",") != "Segunda,Tercera,Primera" {
		t.Errorf("project_id = %q, read id = %q, titles = %v; want the canonical id and alta, media, baja", body.ProjectID, lister.lastID, titles)
	}
}

func TestBacklogCompositionMapsCollectionGetErrors(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		err       error
		status    int
		reads     int
		code      string
	}{
		{"unknown project", updateProjectID, storyapplication.ErrProjectNotFound, http.StatusNotFound, 1, `"error":"project_not_found"`},
		{"malformed identifier", "not-a-uuid", nil, http.StatusUnprocessableEntity, 0, `"error":"validation_failed"`},
		{"unexpected failure", updateProjectID, errors.New("connection refused: host=db"), http.StatusInternalServerError, 1, `"error":"internal_error"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lister := &fakeStoryLister{err: tt.err}
			handler := newBacklogHandler(&fakeStoryRepository{}, &fakeStoryUpdater{}, lister)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/projects/"+tt.projectID+"/stories", nil))
			if response.Code != tt.status || lister.calls != tt.reads || !strings.Contains(response.Body.String(), tt.code) ||
				strings.Contains(response.Body.String(), `"stories"`) || strings.Contains(response.Body.String(), "connection refused") {
				t.Errorf("GET = %d (reads %d) %s; want %d with %d reads, %s and no stories or internal detail",
					response.Code, lister.calls, response.Body.String(), tt.status, tt.reads, tt.code)
			}
		})
	}
}

func TestBacklogCompositionKeepsTheRouteBoundaries(t *testing.T) {
	collection := "/projects/" + updateProjectID + "/stories"
	item := collection + "/" + updateStoryID
	creationBody := `{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Listo"]}`
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
		reads  int
		writes int
	}{
		{"collection HEAD", http.MethodHead, collection, "", http.StatusOK, 1, 0},
		{"collection POST", http.MethodPost, collection, creationBody, http.StatusCreated, 0, 1},
		{"collection PUT", http.MethodPut, collection, updateBody, http.StatusMethodNotAllowed, 0, 0},
		{"collection DELETE", http.MethodDelete, collection, "", http.StatusMethodNotAllowed, 0, 0},
		{"collection PATCH", http.MethodPatch, collection, "", http.StatusMethodNotAllowed, 0, 0},
		{"item GET", http.MethodGet, item, "", http.StatusMethodNotAllowed, 0, 0},
		{"trailing slash GET", http.MethodGet, collection + "/", "", http.StatusNotFound, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stories, updater, lister := &fakeStoryRepository{}, &fakeStoryUpdater{}, &fakeStoryLister{stories: backlogStories()}
			handler := newBacklogHandler(stories, updater, lister)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if response.Code != tt.status || lister.calls != tt.reads || len(stories.stories) != tt.writes || len(updater.updates) != 0 {
				t.Errorf("%s %s = %d, reads = %d, creations = %d, updates = %d; want %d, %d reads and %d creations",
					tt.method, tt.path, response.Code, lister.calls, len(stories.stories), len(updater.updates), tt.status, tt.reads, tt.writes)
			}
		})
	}
}

func TestBacklogCompositionKeepsStoryUpdateWorking(t *testing.T) {
	updater := &fakeStoryUpdater{}
	handler := newBacklogHandler(&fakeStoryRepository{}, updater, &fakeStoryLister{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut,
		"/projects/"+updateProjectID+"/stories/"+updateStoryID, strings.NewReader(updateBody)))
	if response.Code != http.StatusOK || len(updater.updates) != 1 {
		t.Errorf("PUT status = %d, updates = %d; body = %s", response.Code, len(updater.updates), response.Body.String())
	}
}

// TestCompositionWithoutListerDoesNotExposeTheBacklog covers the schema versions below 4 (creation
// and update, or creation only): the collection route only admits POST, so GET stays 405.
func TestCompositionWithoutListerDoesNotExposeTheBacklog(t *testing.T) {
	collection := "/projects/" + updateProjectID + "/stories"
	for name, updater := range map[string]*fakeStoryUpdater{"creation and update": {}, "creation only": nil} {
		t.Run(name, func(t *testing.T) {
			handler := newStoryHandler(&fakeStoryRepository{}, updater)
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(method, collection, nil))
				if response.Code != http.StatusMethodNotAllowed || strings.Contains(response.Body.String(), "project_id") {
					t.Errorf("%s without a lister = %d %s, want 405 without a backlog", method, response.Code, response.Body.String())
				}
			}
		})
	}

	projectsOnly := api.NewHTTPHandler(&fakeProjectRepository{}, func() string { return updateProjectID })
	response := httptest.NewRecorder()
	projectsOnly.ServeHTTP(response, httptest.NewRequest(http.MethodGet, collection, nil))
	if response.Code != http.StatusNotFound {
		t.Errorf("project-only GET = %d, want 404 because no story route exists", response.Code)
	}
}

func TestNewHTTPHandlerRegistersStoryUpdateRouteWhenUpdaterIsPresent(t *testing.T) {
	updater := &fakeStoryUpdater{}
	handler := newStoryHandler(&fakeStoryRepository{}, updater)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut,
		"/projects/"+updateProjectID+"/stories/"+updateStoryID, strings.NewReader(updateBody)))
	if response.Code != http.StatusOK || len(updater.updates) != 1 {
		t.Fatalf("PUT status = %d, updates = %d; body = %s", response.Code, len(updater.updates), response.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["id"] != updateStoryID || result["project_id"] != updateProjectID || result["status"] != "en_progreso" ||
		result["estimated_hours"] != 2.5 || result["story_points"] != nil {
		t.Errorf("updated story response = %v", result)
	}
}

func TestNewHTTPHandlerWithoutUpdaterDoesNotExposeStoryUpdate(t *testing.T) {
	stories := &fakeStoryRepository{}
	handler := newStoryHandler(stories, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut,
		"/projects/"+updateProjectID+"/stories/"+updateStoryID, strings.NewReader(updateBody)))
	if response.Code != http.StatusNotFound {
		t.Errorf("PUT without updater status = %d, want 404", response.Code)
	}

	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/projects/"+updateProjectID+"/stories",
		strings.NewReader(`{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Listo"]}`)))
	if created.Code != http.StatusCreated || len(stories.stories) != 1 {
		t.Errorf("creation status = %d, writes = %d; body = %s", created.Code, len(stories.stories), created.Body.String())
	}
}

func TestStoryRoutingPreservesCollectionAndItemBoundaries(t *testing.T) {
	// No lister is composed here (schema below version 4), so GET on the collection stays 405.
	updater := &fakeStoryUpdater{}
	handler := newStoryHandler(&fakeStoryRepository{}, updater)
	collection := "/projects/" + updateProjectID + "/stories"
	item := collection + "/" + updateStoryID
	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"collection GET", http.MethodGet, collection, http.StatusMethodNotAllowed},
		{"collection PUT", http.MethodPut, collection, http.StatusMethodNotAllowed},
		{"collection DELETE", http.MethodDelete, collection, http.StatusMethodNotAllowed},
		{"trailing slash", http.MethodPut, collection + "/", http.StatusNotFound},
		{"item GET", http.MethodGet, item, http.StatusMethodNotAllowed},
		{"item POST", http.MethodPost, item, http.StatusMethodNotAllowed},
		{"item PATCH", http.MethodPatch, item, http.StatusMethodNotAllowed},
		{"item DELETE", http.MethodDelete, item, http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, strings.NewReader(updateBody)))
			if response.Code != tt.status || len(updater.updates) != 0 {
				t.Errorf("%s %s status = %d, updates = %d; want %d and 0 updates", tt.method, tt.path, response.Code, len(updater.updates), tt.status)
			}
		})
	}
}

func TestStoryUpdateTwiceKeepsTheLastWrite(t *testing.T) {
	updater := &fakeStoryUpdater{}
	handler := newStoryHandler(&fakeStoryRepository{}, updater)
	path := "/projects/" + updateProjectID + "/stories/" + updateStoryID
	for _, body := range []string{updateBody, strings.Replace(updateBody, `"status":"en_progreso"`, `"status":"completada"`, 1)} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, path, strings.NewReader(body)))
		if response.Code != http.StatusOK {
			t.Fatalf("PUT status = %d; body = %s", response.Code, response.Body.String())
		}
	}
	if len(updater.updates) != 2 || updater.updates[0].Status != "en_progreso" || updater.updates[1].Status != "completada" {
		t.Errorf("updates = %+v, want en_progreso then completada", updater.updates)
	}
}

func TestNewHTTPHandlerRegistersStoryAndPreservesProjects(t *testing.T) {
	projects := &fakeProjectRepository{}
	stories := &fakeStoryRepository{}
	handler := api.NewHTTPHandler(projects, func() string { return "5c21cbd4-d9a7-42df-9c3a-c0866f058746" },
		api.StoryDependencies{Repository: stories, GenerateID: func() string { return "e99c05a4-03ea-4c19-8f59-286dd59e7aa1" }})
	projectID := "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	body := `{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Primero","Segundo"]}`
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/stories", strings.NewReader(body)))
	if created.Code != http.StatusCreated || len(stories.stories) != 1 {
		t.Fatalf("story status = %d, writes = %d; body = %s", created.Code, len(stories.stories), created.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["project_id"] != projectID || result["story_points"] != nil || result["status"] != "pendiente" {
		t.Errorf("story response = %v", result)
	}

	// No lister is composed here (schema below version 4), so GET on the collection stays 405.
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/projects/"+projectID+"/stories", strings.NewReader(body)))
		if response.Code != http.StatusMethodNotAllowed || len(stories.stories) != 1 {
			t.Errorf("%s status = %d, writes = %d", method, response.Code, len(stories.stories))
		}
	}

	project := httptest.NewRecorder()
	handler.ServeHTTP(project, httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`)))
	if project.Code != http.StatusCreated || len(projects.projects) != 1 {
		t.Errorf("project status = %d, writes = %d; body = %s", project.Code, len(projects.projects), project.Body.String())
	}
}

func TestProjectOnlyCompositionDoesNotExposeStoryRoute(t *testing.T) {
	projects := &fakeProjectRepository{}
	handler := api.NewHTTPHandler(projects, func() string { return "5c21cbd4-d9a7-42df-9c3a-c0866f058746" })
	storyRequest := httptest.NewRequest(http.MethodPost, "/projects/5c21cbd4-d9a7-42df-9c3a-c0866f058746/stories",
		strings.NewReader(`{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Listo"]}`))
	storyResponse := httptest.NewRecorder()
	handler.ServeHTTP(storyResponse, storyRequest)
	if storyResponse.Code != http.StatusNotFound {
		t.Errorf("project-only story status = %d, want 404", storyResponse.Code)
	}

	projectResponse := httptest.NewRecorder()
	handler.ServeHTTP(projectResponse, httptest.NewRequest(http.MethodPost, "/projects",
		strings.NewReader(`{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`)))
	if projectResponse.Code != http.StatusCreated || len(projects.projects) != 1 {
		t.Errorf("project-only POST status = %d, writes = %d", projectResponse.Code, len(projects.projects))
	}
}

func TestLoadConfigRequiresDatabaseURL(t *testing.T) {
	_, err := api.LoadConfig(func(string) string { return "" })

	if err == nil {
		t.Fatal("loadConfig() error = nil, want missing DATABASE_URL error")
	}
}

func TestLoadConfigUsesDatabaseURLAndDefaultAddress(t *testing.T) {
	config, err := api.LoadConfig(func(key string) string {
		if key == "DATABASE_URL" {
			return "postgres://postgres:postgres@localhost:5432/projects?sslmode=disable"
		}
		return ""
	})

	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config.DatabaseURL == "" {
		t.Fatal("DatabaseURL is empty")
	}
	if config.Address != ":8080" {
		t.Errorf("Address = %q, want :8080", config.Address)
	}
}

func TestLoadConfigUsesConfiguredAddress(t *testing.T) {
	config, err := api.LoadConfig(func(key string) string {
		switch key {
		case "DATABASE_URL":
			return "postgres://postgres:postgres@localhost:5432/projects?sslmode=disable"
		case "HTTP_ADDR":
			return ":9090"
		default:
			return ""
		}
	})

	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config.Address != ":9090" {
		t.Errorf("Address = %q, want :9090", config.Address)
	}
}

func TestNewProjectIDReturnsUUID(t *testing.T) {
	id := api.NewProjectID()

	if len(id) != 36 {
		t.Fatalf("newProjectID() = %q, want UUID string", id)
	}
}

func TestNewHTTPHandlerRegistersCreateProjectRoute(t *testing.T) {
	repository := &fakeProjectRepository{}
	handler := api.NewHTTPHandler(repository, func() string { return "5c21cbd4-d9a7-42df-9c3a-c0866f058746" })
	request := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{
		"name":"Metrics portal",
		"start_date":"2026-03-01",
		"planned_finish_date":"2026-06-30"
	}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["id"] != "5c21cbd4-d9a7-42df-9c3a-c0866f058746" {
		t.Errorf("id = %v, want generated UUID", body["id"])
	}
	if len(repository.projects) != 1 {
		t.Fatalf("repository projects = %d, want 1", len(repository.projects))
	}
}

func TestNewHTTPHandlerRegistersProjectUpdateRoute(t *testing.T) {
	repository := &fakeProjectRepository{}
	handler := api.NewHTTPHandler(repository, func() string { return "5c21cbd4-d9a7-42df-9c3a-c0866f058746" })
	request := httptest.NewRequest(http.MethodPut, "/projects/5c21cbd4-d9a7-42df-9c3a-c0866f058746",
		strings.NewReader(`{"name":"Updated","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(repository.projects) != 1 {
		t.Fatalf("PUT status = %d, writes = %d; body=%s", response.Code, len(repository.projects), response.Body.String())
	}
	if repository.projects[0].ID != "5c21cbd4-d9a7-42df-9c3a-c0866f058746" || repository.projects[0].Name != "Updated" {
		t.Errorf("updated project = %#v", repository.projects[0])
	}
}

func TestNewHTTPHandlerMapsInvalidAndUnknownUpdateIDs(t *testing.T) {
	tests := []struct {
		name            string
		projectID       string
		repositoryError error
		status          int
		updateCalls     int
	}{
		{name: "malformed UUID", projectID: "not-a-uuid", status: http.StatusUnprocessableEntity},
		{name: "unknown project", projectID: "5c21cbd4-d9a7-42df-9c3a-c0866f058746", repositoryError: application.ErrProjectNotFound, status: http.StatusNotFound, updateCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeProjectRepository{updateErr: tt.repositoryError}
			handler := api.NewHTTPHandler(repository, func() string { return "generated-id" })
			request := httptest.NewRequest(http.MethodPut, "/projects/"+tt.projectID,
				strings.NewReader(`{"name":"Updated","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}`))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("PUT status = %d, want %d; body=%s", response.Code, tt.status, response.Body.String())
			}
			if repository.updateCalls != tt.updateCalls || len(repository.projects) != 0 {
				t.Errorf("update calls = %d, project writes = %d; want calls=%d, writes=0", repository.updateCalls, len(repository.projects), tt.updateCalls)
			}
		})
	}
}
