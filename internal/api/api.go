// Package api provides bootstrap helpers for the HTTP API.
package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/project/transport/http"
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

// NewHTTPHandler builds the HTTP handler for project endpoints.
func NewHTTPHandler(repository application.ProjectRepository, generateID application.IDGenerator) http.Handler {
	useCase := application.NewCreateProjectUseCase(repository, generateID)
	mux := http.NewServeMux()
	mux.Handle("POST /projects", transporthttp.NewCreateProjectHandler(useCase))
	return mux
}
