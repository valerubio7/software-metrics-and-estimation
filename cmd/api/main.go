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
	sprintpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/infrastructure/postgres"
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
	readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr)
	projects := projectpostgres.NewPostgresProjectRepository(pool)
	dependencies := api.HTTPDependencies{}
	if readiness.Stories {
		stories := storypostgres.NewPostgresStoryRepository(pool)
		storyDependencies := &api.StoryDependencies{Repository: stories, GenerateID: api.NewProjectID}
		if version >= 3 {
			storyDependencies.Updater = stories
		}
		if version >= 4 {
			storyDependencies.Lister = stories
		}
		if readiness.Assignment {
			storyDependencies.Assigner = stories
		}
		dependencies.Stories = storyDependencies
		switch version {
		case 2:
			log.Printf("story creation available; story update unavailable until migration 000003 is clean (version=%d)", version)
		case 3:
			log.Printf("story creation and update available; story backlog unavailable until migration 000004 is clean (version=%d)", version)
		default:
			log.Printf("story creation, update and backlog available (version=%d)", version)
		}
	} else {
		log.Printf("story creation unavailable until migration 000002 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
	}
	if readiness.Sprints {
		dependencies.Sprints = &api.SprintDependencies{Repository: sprintpostgres.NewPostgresSprintRepository(pool), GenerateID: api.NewProjectID}
	} else {
		log.Printf("sprint creation unavailable until migration 000003 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
	}
	handler := api.NewHTTPHandlerWithDependencies(projects, api.NewProjectID, dependencies)
	server := &http.Server{Addr: config.Address, Handler: handler}

	log.Printf("API listening on %s", config.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}
