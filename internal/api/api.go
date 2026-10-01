// Package api provides bootstrap helpers for the HTTP API.
package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/project/transport/http"
	sprintapplication "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	sprinthttp "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/transport/http"
	storyapplication "github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storyhttp "github.com/valerubio7/software-metrics-and-estimation/internal/story/transport/http"
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
	Repository storyapplication.StoryRepository
	GenerateID storyapplication.IDGenerator
	Updater    storyapplication.StoryUpdater
	Lister     storyapplication.StoryLister // nil: GET on the collection is not registered (the mux answers 405)
	Assigner   storyapplication.StorySprintAssigner
}

// SprintDependencies enables sprint creation after migration 000003 is clean.
type SprintDependencies struct {
	Repository sprintapplication.SprintRepository
	GenerateID sprintapplication.IDGenerator
}

// HTTPDependencies contains optional migration-backed API modules.
type HTTPDependencies struct {
	Stories *StoryDependencies
	Sprints *SprintDependencies
}

// MigrationReadiness describes which handlers are safe to compose for a schema state.
type MigrationReadiness struct {
	Projects   bool
	Stories    bool
	Sprints    bool
	Assignment bool
}

// ResolveMigrationReadiness keeps project creation available while gating schema-backed routes.
func ResolveMigrationReadiness(version int, dirty bool, lookupErr error) MigrationReadiness {
	readiness := MigrationReadiness{Projects: true}
	if lookupErr != nil || dirty {
		return readiness
	}
	readiness.Stories = version >= 2
	readiness.Sprints = version >= 3
	readiness.Assignment = version >= 6
	return readiness
}

// NewHTTPHandler builds the HTTP handler; existing project-only callers remain valid.
// The caller must verify migration 000002 before supplying story dependencies,
// migration 000003 before supplying StoryDependencies.Updater and migration 000004
// before supplying StoryDependencies.Lister.
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
	mux := http.NewServeMux()
	mux.Handle("POST /projects", transporthttp.NewCreateProjectHandler(useCase))
	mux.Handle("PUT /projects/{project_id}", transporthttp.NewUpdateProjectHandler(updateUseCase))
	if dependencies.Stories != nil {
		stories := dependencies.Stories
		storyUseCase := storyapplication.NewCreateStoryUseCase(stories.Repository, stories.GenerateID)
		mux.Handle("POST /projects/{project_id}/stories", storyhttp.NewCreateStoryHandler(storyUseCase))
		if stories.Updater != nil {
			updateStoryUseCase := storyapplication.NewUpdateStoryUseCase(stories.Updater)
			mux.Handle("PUT /projects/{project_id}/stories/{story_id}", storyhttp.NewUpdateStoryHandler(updateStoryUseCase))
		}
		if stories.Lister != nil {
			listStoriesUseCase := storyapplication.NewListStoriesUseCase(stories.Lister)
			mux.Handle("GET /projects/{project_id}/stories", storyhttp.NewListStoriesHandler(listStoriesUseCase))
		}
		if stories.Assigner != nil {
			assignUseCase := storyapplication.NewAssignStoriesUseCase(stories.Assigner)
			mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories", storyhttp.NewAssignStoriesHandler(assignUseCase))
		}
	}
	if dependencies.Sprints != nil {
		sprintUseCase := sprintapplication.NewCreateSprintUseCase(dependencies.Sprints.Repository, dependencies.Sprints.GenerateID)
		mux.Handle("POST /projects/{project_id}/sprints", sprinthttp.NewCreateSprintHandler(sprintUseCase))
	}
	return mux
}
