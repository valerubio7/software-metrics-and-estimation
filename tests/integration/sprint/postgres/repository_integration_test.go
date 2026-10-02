package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
	sprintpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/infrastructure/postgres"
)

const (
	projectID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	sprintID  = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
)

func TestSprintClosedMigrationDefaultDownAndUp(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)

	var columnDefault string
	if err := pool.QueryRow(context.Background(), `SELECT column_default FROM information_schema.columns WHERE table_name = 'sprints' AND column_name = 'is_closed'`).Scan(&columnDefault); err != nil {
		t.Fatalf("read is_closed default after migration: %v", err)
	}
	if columnDefault != "false" {
		t.Fatalf("is_closed default = %q, want false", columnDefault)
	}
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Default')", sprintID, projectID); err != nil {
		t.Fatalf("insert sprint with default closed state: %v", err)
	}
	var isClosed bool
	if err := pool.QueryRow(context.Background(), "SELECT is_closed FROM sprints WHERE id = $1", sprintID).Scan(&isClosed); err != nil {
		t.Fatal(err)
	}
	if isClosed {
		t.Fatal("new sprint is closed; want false by default")
	}

	applySprintMigration(t, pool, "000006_add_sprint_closed.down.sql")
	var columnExists bool
	if err := pool.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sprints' AND column_name = 'is_closed')`).Scan(&columnExists); err != nil {
		t.Fatal(err)
	}
	if columnExists {
		t.Fatal("is_closed still exists after down migration")
	}
	applySprintMigration(t, pool, "000006_add_sprint_closed.up.sql")
	if err := pool.QueryRow(context.Background(), `SELECT column_default FROM information_schema.columns WHERE table_name = 'sprints' AND column_name = 'is_closed'`).Scan(&columnDefault); err != nil {
		t.Fatalf("read is_closed default after reapplying migration: %v", err)
	}
	if columnDefault != "false" {
		t.Fatalf("is_closed default after reapplying migration = %q, want false", columnDefault)
	}
}

func applySprintMigration(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	migration, err := os.ReadFile(filepath.Join(sprintModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", name))
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply migration %s: %v", name, err)
	}
}

func TestSprintRepositoryPersistsSprintFields(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	sprint := domain.Sprint{ID: sprintID, ProjectID: projectID, SprintGoal: "  Deliver the first metrics flow  "}

	if err := sprintpostgres.NewPostgresSprintRepository(pool).Create(context.Background(), sprint); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var id, storedProjectID, goal string
	err := pool.QueryRow(context.Background(), `
		SELECT id::text, project_id::text, sprint_goal FROM sprints WHERE id = $1
	`, sprint.ID).Scan(&id, &storedProjectID, &goal)
	if err != nil {
		t.Fatalf("read persisted sprint: %v", err)
	}
	if id != sprint.ID || storedProjectID != sprint.ProjectID || goal != sprint.SprintGoal {
		t.Errorf("persisted sprint = (%q, %q, %q), want (%q, %q, %q)",
			id, storedProjectID, goal, sprint.ID, sprint.ProjectID, sprint.SprintGoal)
	}
	if got := sprintCount(t, pool); got != 1 {
		t.Errorf("sprint count = %d, want 1", got)
	}
}

func TestSprintRepositoryMapsOnlyNamedProjectForeignKey(t *testing.T) {
	pool := sprintDatabase(t)
	repository := sprintpostgres.NewPostgresSprintRepository(pool)
	missingProjectSprint := domain.Sprint{ID: sprintID, ProjectID: projectID, SprintGoal: "Missing project"}

	var constraintDefinition string
	err := pool.QueryRow(context.Background(), `
		SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'sprints'::regclass AND conname = 'sprints_project_id_fkey'
	`).Scan(&constraintDefinition)
	if err != nil {
		t.Fatalf("named project FK missing: %v", err)
	}
	if !strings.Contains(constraintDefinition, "REFERENCES projects(id) ON DELETE RESTRICT") {
		t.Errorf("project FK = %q, want projects(id) ON DELETE RESTRICT", constraintDefinition)
	}

	err = repository.Create(context.Background(), missingProjectSprint)
	if !errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("Create(missing project) error = %v, want ErrProjectNotFound", err)
	}
	if got := sprintCount(t, pool); got != 0 {
		t.Errorf("sprint count after missing project = %d, want 0", got)
	}

	insertProject(t, pool)
	validSprint := domain.Sprint{ID: sprintID, ProjectID: projectID, SprintGoal: "Persist once"}
	if err := repository.Create(context.Background(), validSprint); err != nil {
		t.Fatalf("seed sprint: %v", err)
	}
	err = repository.Create(context.Background(), validSprint)
	assertDatabaseError(t, err, "23505", "sprints_pkey")
	if errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("primary-key violation mislabeled as missing project: %v", err)
	}
	if got := sprintCount(t, pool); got != 1 {
		t.Errorf("sprint count after failed duplicate insert = %d, want 1", got)
	}
}

func sprintDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("sprints_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL container (Docker must be available): %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate PostgreSQL container: %v", err)
		}
	})
	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get PostgreSQL connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	deadline := time.Now().Add(10 * time.Second)
	for {
		pingCtx, cancel := context.WithTimeout(ctx, time.Second)
		err = pool.Ping(pingCtx)
		cancel()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("wait for PostgreSQL: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	for _, name := range []string{"000001_create_projects.up.sql", "000002_create_stories.up.sql", "000003_create_sprints.up.sql", "000006_add_sprint_closed.up.sql"} {
		migration, err := os.ReadFile(filepath.Join(sprintModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
	return pool
}

func insertProject(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO projects (id, name, start_date, planned_finish_date)
		VALUES ($1, 'Metrics', '2026-03-01', '2026-06-30')
	`, projectID)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
}

func assertDatabaseError(t *testing.T, err error, code, constraint string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error = %v, want PostgreSQL %s on %s", err, code, constraint)
	}
	if pgErr.Code != code || pgErr.ConstraintName != constraint {
		t.Errorf("database error = (%s, %s), want (%s, %s)", pgErr.Code, pgErr.ConstraintName, code, constraint)
	}
}

func sprintCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprints").Scan(&count); err != nil {
		t.Fatalf("count sprints: %v", err)
	}
	return count
}

func sprintModuleRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("locate module root: go.mod not found")
		}
		directory = parent
	}
}
