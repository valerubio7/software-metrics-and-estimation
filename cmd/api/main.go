package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-estimation/internal/project/infrastructure/postgres"
	transporthttp "github.com/valerubio7/software-metrics-estimation/internal/project/transport/http"
)

type config struct {
	DatabaseURL string
	Address     string
}

func main() {
	config, err := loadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping PostgreSQL: %v", err)
	}

	handler := newHTTPHandler(postgres.NewPostgresProjectRepository(pool), newProjectID)
	server := &http.Server{Addr: config.Address, Handler: handler}

	log.Printf("API listening on %s", config.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}

func loadConfig(getenv func(string) string) (config, error) {
	databaseURL := getenv("DATABASE_URL")
	if databaseURL == "" {
		return config{}, fmt.Errorf("DATABASE_URL is required")
	}

	address := getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}

	return config{DatabaseURL: databaseURL, Address: address}, nil
}

func newProjectID() string {
	return uuid.NewString()
}

func newHTTPHandler(repository application.ProjectRepository, generateID application.IDGenerator) http.Handler {
	useCase := application.NewCreateProjectUseCase(repository, generateID)
	mux := http.NewServeMux()
	mux.Handle("POST /projects", transporthttp.NewCreateProjectHandler(useCase))
	return mux
}
