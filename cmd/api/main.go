package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
)

func main() {
	config, err := api.LoadConfig(os.Getenv)
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

	// Only expose stories after the externally managed migration is clean.
	var version int
	var dirty bool
	migrationErr := pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	var handler http.Handler
	if migrationErr == nil && !dirty && version >= 2 {
		handler = api.NewHTTPHandler(projectpostgres.NewPostgresProjectRepository(pool), api.NewProjectID,
			api.StoryDependencies{Repository: storypostgres.NewPostgresStoryRepository(pool), GenerateID: api.NewProjectID})
	} else {
		log.Printf("story creation unavailable until migration 000002 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
		handler = api.NewHTTPHandler(projectpostgres.NewPostgresProjectRepository(pool), api.NewProjectID)
	}
	server := &http.Server{Addr: config.Address, Handler: handler}

	log.Printf("API listening on %s", config.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}
