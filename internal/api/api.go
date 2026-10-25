// Package api provides bootstrap helpers for the HTTP API.
package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	effortapplication "github.com/valerubio7/software-metrics-and-estimation/internal/effort/application"
	efforthttp "github.com/valerubio7/software-metrics-and-estimation/internal/effort/transport/http"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/project/transport/http"
	memberapplication "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	memberhttp "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/transport/http"
	sprintapplication "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	sprinthttp "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/transport/http"
	storyapplication "github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storyhttp "github.com/valerubio7/software-metrics-and-estimation/internal/story/transport/http"
	taskapplication "github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	taskhttp "github.com/valerubio7/software-metrics-and-estimation/internal/task/transport/http"
)

// Config contains the API runtime configuration.
type Config struct {
	DatabaseURL string
	Address     string
}

// LoadConfig reads API configuration from the supplied environment lookup.
func LoadConfig(getenv func(string) string) (Config, error) {
	databaseURL := getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	address := getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}

	return Config{DatabaseURL: databaseURL, Address: address}, nil
}

// NewProjectID generates a UUID string for a new project.
func NewProjectID() string {
	return uuid.NewString()
}

// StoryDependencies enables story creation after the story migration is available.
// Updater and Lister are optional: the update route is registered only when Updater is
// supplied and the backlog query only when Lister is supplied, which the caller does
// after verifying migrations 000003 and 000004 respectively.
type StoryDependencies struct {
	Repository        storyapplication.StoryRepository
	GenerateID        storyapplication.IDGenerator
	Updater           storyapplication.StoryUpdater
	StoryPointsSetter storyapplication.StoryPointsSetter
	Lister            storyapplication.StoryLister               // nil: GET on the collection is not registered (the mux answers 405)
	Assigner          storyapplication.StorySprintAssigner       // supplied only after migration 000008 is clean
	AssignedLister    storyapplication.AssignedSprintStoryReader // supplied only after migration 000008 is clean
}

// SprintDependencies enables sprint creation after reconciliation migration 000005 is clean.
// Closer and Lister are optional: closure is registered only when Closer is supplied (migration 000008)
// and the closed-sprint history only when Lister is supplied (migration 000011), independently of each other.
type SprintDependencies struct {
	Repository sprintapplication.SprintRepository
	GenerateID sprintapplication.IDGenerator
	Closer     sprintapplication.SprintCloser        // supplied only after migration 000008 is clean
	Lister     sprintapplication.SprintHistoryReader // supplied only after migration 000011 is clean; nil: GET is not registered
	OpenLister sprintapplication.OpenSprintReader    // supplied only after migration 000011 is clean; nil: GET is not registered
}

// MemberDependencies enables project-member registration after migration v6 is clean.
// Lister is optional: the member collection is readable only when it is supplied.
type MemberDependencies struct {
	Repository memberapplication.MemberRepository
	GenerateID memberapplication.IDGenerator
	Lister     memberapplication.MemberLister
}

// TaskDependencies enables task creation after migration 000009 is clean.
// Lister is optional: story task listing is registered only when it is supplied.
type TaskDependencies struct {
	Repository taskapplication.TaskRepository
	GenerateID taskapplication.IDGenerator
	Lister     taskapplication.TaskLister
}

// CompletionDependencies enables story completion within a sprint after migration 000010 is clean.
type CompletionDependencies struct {
	Completer storyapplication.SprintStoryCompleter
}

// EffortDependencies enables effort registration after migration 000012 is clean.
// Now is optional: when nil the route reads the current UTC time.
type EffortDependencies struct {
	Repository effortapplication.EffortRepository
	GenerateID effortapplication.IDGenerator
	Now        effortapplication.Clock
}

// HTTPDependencies contains optional migration-backed API modules.
type HTTPDependencies struct {
	Stories *StoryDependencies
	Sprints *SprintDependencies
	Members *MemberDependencies
	Tasks   *TaskDependencies // nil: the task route is not registered (the mux answers 404)
	// Completion nil: the completion route is not registered (the mux answers 404).
	Completion *CompletionDependencies
	// Effort nil: the effort registration route is not registered (the mux answers 404).
	Effort *EffortDependencies
}

// MigrationReadiness describes which handlers are safe to compose for a schema state.
type MigrationReadiness struct {
	Projects    bool
	Stories     bool
	StoryPoints bool // story_points exists from schema version 2, independently of generic story updates
	Sprints     bool
	Members     bool
	Assignment  bool
	SprintClose bool
	Tasks       bool
	Completion  bool
	// SprintHistory is true for a clean schema at version 11 or later (sprints.seq from migration 000011).
	SprintHistory bool
	// Effort is true for a clean schema at version 12 or later (effort_records from migration 000012).
	Effort bool
}

// ResolveMigrationReadiness keeps project creation available while gating schema-backed routes.
func ResolveMigrationReadiness(version int, dirty bool, lookupErr error) MigrationReadiness {
	readiness := MigrationReadiness{Projects: true}
	if lookupErr != nil || dirty {
		return readiness
	}
	readiness.Stories = version >= 2
	readiness.StoryPoints = version >= 2
	readiness.Sprints = version >= 5
	readiness.Members = version >= 6
	readiness.Assignment = version >= 8
	readiness.SprintClose = version >= 8
	readiness.Tasks = version >= 9
	readiness.Completion = version >= 10
	readiness.SprintHistory = version >= 11
	readiness.Effort = version >= 12
	return readiness
}

// NewHTTPHandler builds the HTTP handler; existing project-only callers remain valid.
// The caller must verify migration 000002 before supplying story dependencies or
// StoryDependencies.StoryPointsSetter, migration 000003 before supplying Updater,
// and migration 000004 before supplying Lister.
func NewHTTPHandler(repository application.ProjectRepository, generateID application.IDGenerator, stories ...StoryDependencies) http.Handler {
	dependencies := HTTPDependencies{}
	if len(stories) > 0 {
		dependencies.Stories = &stories[0]
	}
	return NewHTTPHandlerWithDependencies(repository, generateID, dependencies)
}

// NewHTTPHandlerWithDependencies adds optional feature routes while preserving the legacy constructor.
func NewHTTPHandlerWithDependencies(repository application.ProjectRepository, generateID application.IDGenerator, dependencies HTTPDependencies) http.Handler {
	useCase := application.NewCreateProjectUseCase(repository, generateID)
	updateUseCase := application.NewUpdateProjectUseCase(repository)
	statusReader, supportsProjectReads := repository.(application.ProjectReader)
	mux := http.NewServeMux()
	mux.Handle("POST /projects", transporthttp.NewCreateProjectHandler(useCase))
	mux.Handle("PUT /projects/{project_id}", transporthttp.NewUpdateProjectHandler(updateUseCase))
	if supportsProjectReads {
		statusUseCase := application.NewGetProjectStatusUseCase(statusReader)
		mux.Handle("GET /projects/{project_id}", transporthttp.NewGetProjectStatusHandler(statusUseCase, func() time.Time { return time.Now().UTC() }))
	}
	if projectLister, supportsProjectList := repository.(application.ProjectLister); supportsProjectList {
		listUseCase := application.NewListProjectsUseCase(projectLister)
		mux.Handle("GET /projects", transporthttp.NewListProjectsHandler(listUseCase, func() time.Time { return time.Now().UTC() }))
	}
	if dependencies.Stories != nil {
		storyUseCase := storyapplication.NewCreateStoryUseCase(dependencies.Stories.Repository, dependencies.Stories.GenerateID)
		mux.Handle("POST /projects/{project_id}/stories", storyhttp.NewCreateStoryHandler(storyUseCase))
		if dependencies.Stories.StoryPointsSetter != nil {
			setPointsUseCase := storyapplication.NewSetStoryPointsUseCase(dependencies.Stories.StoryPointsSetter)
			mux.Handle("PUT /projects/{project_id}/stories/{story_id}/story-points", storyhttp.NewSetStoryPointsHandler(setPointsUseCase))
		}
		if dependencies.Stories.Updater != nil {
			updateStoryUseCase := storyapplication.NewUpdateStoryUseCase(dependencies.Stories.Updater)
			mux.Handle("PUT /projects/{project_id}/stories/{story_id}", storyhttp.NewUpdateStoryHandler(updateStoryUseCase))
		}
		if dependencies.Stories.Lister != nil {
			listStoriesUseCase := storyapplication.NewListStoriesUseCase(dependencies.Stories.Lister)
			mux.Handle("GET /projects/{project_id}/stories", storyhttp.NewListStoriesHandler(listStoriesUseCase))
		}
	}
	if dependencies.Stories != nil && dependencies.Stories.Assigner != nil {
		assignUseCase := storyapplication.NewAssignStoriesUseCase(dependencies.Stories.Assigner)
		mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories", storyhttp.NewAssignStoriesHandler(assignUseCase))
	}
	if dependencies.Stories != nil && dependencies.Stories.AssignedLister != nil {
		listAssignedUseCase := storyapplication.NewListAssignedSprintStoriesUseCase(dependencies.Stories.AssignedLister)
		mux.Handle("GET /projects/{project_id}/sprints/{sprint_id}/stories", storyhttp.NewListAssignedSprintStoriesHandler(listAssignedUseCase))
	}
	if dependencies.Sprints != nil {
		sprintUseCase := sprintapplication.NewCreateSprintUseCase(dependencies.Sprints.Repository, dependencies.Sprints.GenerateID)
		mux.Handle("POST /projects/{project_id}/sprints", sprinthttp.NewCreateSprintHandler(sprintUseCase))
		if dependencies.Sprints.Closer != nil {
			closeUseCase := sprintapplication.NewCloseSprintUseCase(dependencies.Sprints.Closer)
			mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/closure", sprinthttp.NewCloseSprintHandler(closeUseCase))
		}
		if dependencies.Sprints.Lister != nil {
			listSprintsUseCase := sprintapplication.NewListSprintsUseCase(dependencies.Sprints.Lister)
			mux.Handle("GET /projects/{project_id}/sprints", sprinthttp.NewListSprintsHandler(listSprintsUseCase))
		}
		if dependencies.Sprints.OpenLister != nil {
			listOpenSprintsUseCase := sprintapplication.NewListOpenSprintsUseCase(dependencies.Sprints.OpenLister)
			mux.Handle("GET /projects/{project_id}/sprints/open", sprinthttp.NewListOpenSprintsHandler(listOpenSprintsUseCase))
		}
	}
	if dependencies.Members != nil {
		membersUseCase := memberapplication.NewRegisterMembersUseCase(dependencies.Members.Repository, dependencies.Members.GenerateID)
		mux.Handle("POST /projects/{project_id}/members", memberhttp.NewRegisterMembersHandler(membersUseCase))
		if dependencies.Members.Lister != nil {
			listMembersUseCase := memberapplication.NewListMembersUseCase(dependencies.Members.Lister)
			mux.Handle("GET /projects/{project_id}/members", memberhttp.NewListMembersHandler(listMembersUseCase))
		}
	}
	if dependencies.Tasks != nil {
		createTasksUseCase := taskapplication.NewCreateTasksUseCase(dependencies.Tasks.Repository, dependencies.Tasks.GenerateID)
		mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks", taskhttp.NewCreateTasksHandler(createTasksUseCase))
		if dependencies.Tasks.Lister != nil {
			listTasksUseCase := taskapplication.NewListTasksUseCase(dependencies.Tasks.Lister)
			mux.Handle("GET /projects/{project_id}/stories/{story_id}/tasks", taskhttp.NewListTasksHandler(listTasksUseCase))
		}
	}
	if dependencies.Completion != nil {
		completeUseCase := storyapplication.NewCompleteSprintStoryUseCase(dependencies.Completion.Completer)
		mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion", storyhttp.NewCompleteSprintStoryHandler(completeUseCase))
	}
	if dependencies.Effort != nil {
		now := dependencies.Effort.Now
		if now == nil {
			now = func() time.Time { return time.Now().UTC() }
		}
		registerEffortUseCase := effortapplication.NewRegisterEffortUseCase(dependencies.Effort.Repository, dependencies.Effort.GenerateID, now)
		mux.Handle("POST /projects/{project_id}/stories/{story_id}/effort-records", efforthttp.NewRegisterEffortHandler(registerEffortUseCase))
	}
	return mux
}
