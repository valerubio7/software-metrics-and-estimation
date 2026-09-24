// Package api provides bootstrap helpers for the HTTP API.
package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/project/transport/http"
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
type StoryDependencies struct {
	Repository storyapplication.StoryRepository
	GenerateID storyapplication.IDGenerator
}

// NewHTTPHandler builds the HTTP handler; existing project-only callers remain valid.
// The caller must verify migration 000002 before supplying story dependencies.
func NewHTTPHandler(repository application.ProjectRepository, generateID application.IDGenerator, stories ...StoryDependencies) http.Handler {
	useCase := application.NewCreateProjectUseCase(repository, generateID)
	mux := http.NewServeMux()
	mux.Handle("POST /projects", transporthttp.NewCreateProjectHandler(useCase))
	if len(stories) != 0 {
		storyUseCase := storyapplication.NewCreateStoryUseCase(stories[0].Repository, stories[0].GenerateID)
		mux.Handle("POST /projects/{project_id}/stories", storyhttp.NewCreateStoryHandler(storyUseCase))
	}
	return mux
}
