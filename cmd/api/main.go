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
	memberpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/infrastructure/postgres"
	sprintpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/infrastructure/postgres"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
	taskpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/task/infrastructure/postgres"
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
	// version 4; sprint creation needs reconciled schema version 5. Task creation needs
	// schema version 9. All of them require a clean externally managed migration state.
	var version int
	var dirty bool
	migrationErr := pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	projects := projectpostgres.NewPostgresProjectRepository(pool)

	dependencies := api.HTTPDependencies{}
	switch {
	case migrationErr == nil && !dirty && version >= 4:
		stories := storypostgres.NewPostgresStoryRepository(pool)
		log.Printf("story creation, update and backlog available (schema version=%d)", version)
		dependencies.Stories = &api.StoryDependencies{Repository: stories, GenerateID: api.NewProjectID, Updater: stories, Lister: stories}
	case migrationErr == nil && !dirty && version >= 3:
		stories := storypostgres.NewPostgresStoryRepository(pool)
		log.Printf("story backlog unavailable until migration 000004 is clean (version=%d)", version)
		dependencies.Stories = &api.StoryDependencies{Repository: stories, GenerateID: api.NewProjectID, Updater: stories}
	case migrationErr == nil && !dirty && version >= 2:
		log.Printf("story update unavailable until migration 000003 is clean (version=%d)", version)
		dependencies.Stories = &api.StoryDependencies{Repository: storypostgres.NewPostgresStoryRepository(pool), GenerateID: api.NewProjectID}
	default:
		log.Printf("story creation unavailable until migration 000002 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
	}
	if readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr); readiness.Sprints {
		dependencies.Sprints = &api.SprintDependencies{Repository: sprintpostgres.NewPostgresSprintRepository(pool), GenerateID: api.NewProjectID}
	} else {
		log.Printf("sprint creation unavailable until reconciliation migration 000005 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
	}
	if readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr); readiness.Members {
		dependencies.Members = &api.MemberDependencies{Repository: memberpostgres.NewPostgresMemberRepository(pool), GenerateID: api.NewProjectID}
		log.Printf("project member registration available (schema version=%d)", version)
	} else {
		log.Printf("project member registration unavailable until migration 000006 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
	}
	if readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr); readiness.Assignment {
		dependencies.Stories.Assigner = storypostgres.NewPostgresStoryRepository(pool)
		log.Printf("story assignment available (schema version=%d)", version)
	} else {
		log.Printf("story assignment unavailable until migration 000008 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
	}
	if readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr); readiness.Tasks {
		dependencies.Tasks = &api.TaskDependencies{Repository: taskpostgres.NewPostgresTaskRepository(pool), GenerateID: api.NewProjectID}
		log.Printf("task creation available (schema version=%d)", version)
	} else {
		log.Printf("task creation unavailable until migration 000009 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
	}
	handler := api.NewHTTPHandlerWithDependencies(projects, api.NewProjectID, dependencies)
	server := &http.Server{Addr: config.Address, Handler: handler}

	log.Printf("API listening on %s", config.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}
