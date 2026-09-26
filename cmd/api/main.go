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

	// Story creation needs schema version 2, story update version 3 and the backlog query
	// version 4; all of them require a clean externally managed migration state.
	var version int
	var dirty bool
	migrationErr := pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	projects := projectpostgres.NewPostgresProjectRepository(pool)
	var handler http.Handler
	switch {
	case migrationErr == nil && !dirty && version >= 4:
		stories := storypostgres.NewPostgresStoryRepository(pool)
		log.Printf("story creation, update and backlog available (schema version=%d)", version)
		handler = api.NewHTTPHandler(projects, api.NewProjectID,
			api.StoryDependencies{Repository: stories, GenerateID: api.NewProjectID, Updater: stories, Lister: stories})
	case migrationErr == nil && !dirty && version >= 3:
		stories := storypostgres.NewPostgresStoryRepository(pool)
		log.Printf("story backlog unavailable until migration 000004 is clean (version=%d)", version)
		handler = api.NewHTTPHandler(projects, api.NewProjectID,
			api.StoryDependencies{Repository: stories, GenerateID: api.NewProjectID, Updater: stories})
	case migrationErr == nil && !dirty && version >= 2:
		log.Printf("story update unavailable until migration 000003 is clean (version=%d)", version)
		handler = api.NewHTTPHandler(projects, api.NewProjectID,
			api.StoryDependencies{Repository: storypostgres.NewPostgresStoryRepository(pool), GenerateID: api.NewProjectID})
	default:
		log.Printf("story creation unavailable until migration 000002 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
		handler = api.NewHTTPHandler(projects, api.NewProjectID)
	}
	server := &http.Server{Addr: config.Address, Handler: handler}

	log.Printf("API listening on %s", config.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}
