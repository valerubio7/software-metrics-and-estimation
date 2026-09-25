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
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
)

func TestPostgresProjectRepositoryCreatePersistsProject(t *testing.T) {
	pool := newPostgresPool(t)
	applyProjectsMigration(t, pool)

	project := domain.Project{
		ID:                "dc46073f-51fb-4393-8e80-6b7f84201cc1",
		Name:              "Metrics portal",
		StartDate:         time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
		PlannedFinishDate: time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC),
	}

	repository := projectpostgres.NewPostgresProjectRepository(pool)
	if err := repository.Create(context.Background(), project); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var gotID, gotName string
	var gotStartDate, gotPlannedFinishDate time.Time
	err := pool.QueryRow(context.Background(), `
		SELECT id::text, name, start_date, planned_finish_date
		FROM projects
		WHERE id = $1
	`, project.ID).Scan(&gotID, &gotName, &gotStartDate, &gotPlannedFinishDate)
	if err != nil {
		t.Fatalf("query persisted project: %v", err)
	}

	if gotID != project.ID {
		t.Errorf("id = %q, want %q", gotID, project.ID)
	}
	if gotName != project.Name {
		t.Errorf("name = %q, want %q", gotName, project.Name)
	}
	if gotStartDate.Format("2006-01-02") != "2026-03-01" {
		t.Errorf("start_date = %s, want 2026-03-01", gotStartDate.Format("2006-01-02"))
	}
	if gotPlannedFinishDate.Format("2006-01-02") != "2026-06-30" {
		t.Errorf("planned_finish_date = %s, want 2026-06-30", gotPlannedFinishDate.Format("2006-01-02"))
	}
}

func TestPostgresProjectRepositoryUpdateChangesBasicFieldsOnly(t *testing.T) {
	pool := newPostgresPool(t)
	applyProjectsMigration(t, pool)
	applyStoriesMigration(t, pool)
	projectID := "dc46073f-51fb-4393-8e80-6b7f84201cc1"
	if _, err := pool.Exec(context.Background(), `INSERT INTO projects (id, name, start_date, planned_finish_date) VALUES ($1, 'Original', '2026-01-01', '2026-12-31')`, projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO stories (id, project_id, title, description, priority, status, acceptance_criteria) VALUES ($1, $2, 'Story', 'Details', 'media', 'pendiente', ARRAY['Done'])`, "e99c05a4-03ea-4c19-8f59-286dd59e7aa1", projectID); err != nil {
		t.Fatal(err)
	}

	repository := projectpostgres.NewPostgresProjectRepository(pool)
	updated := domain.Project{ID: projectID, Name: "Updated", StartDate: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC), PlannedFinishDate: time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)}
	if err := repository.Update(context.Background(), updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	var id, name, title, description, priority, status string
	var startDate, finishDate time.Time
	var criteria []string
	if err := pool.QueryRow(context.Background(), `SELECT p.id::text, p.name, p.start_date, p.planned_finish_date, s.title, s.description, s.priority, s.status, s.acceptance_criteria FROM projects p JOIN stories s ON s.project_id = p.id WHERE p.id = $1`, projectID).Scan(&id, &name, &startDate, &finishDate, &title, &description, &priority, &status, &criteria); err != nil {
		t.Fatal(err)
	}
	if id != projectID || name != "Updated" || startDate.Format("2006-01-02") != "2026-03-01" || finishDate.Format("2006-01-02") != "2026-06-30" {
		t.Errorf("project result = %q, %q, %s, %s", id, name, startDate.Format("2006-01-02"), finishDate.Format("2006-01-02"))
	}
	if title != "Story" || description != "Details" || priority != "media" || status != "pendiente" || len(criteria) != 1 || criteria[0] != "Done" {
		t.Errorf("story changed during project update: %q %q %q %q %v", title, description, priority, status, criteria)
	}
	err := repository.Update(context.Background(), domain.Project{ID: "a8de48f0-692e-4f30-bfa3-1fd7d1cbe5dc", Name: "Missing", StartDate: updated.StartDate, PlannedFinishDate: updated.PlannedFinishDate})
	if !errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("Update(missing) error = %v, want ErrProjectNotFound", err)
	}
}

func TestProjectsMigrationRejectsFinishDateBeforeStartDate(t *testing.T) {
	pool := newPostgresPool(t)
	applyProjectsMigration(t, pool)

	_, err := pool.Exec(context.Background(), `
		INSERT INTO projects (id, name, start_date, planned_finish_date)
		VALUES ($1, $2, $3, $4)
	`, "3e96905f-2911-4e3c-b6b9-c88d087a24b1", "Invalid dates", "2026-03-02", "2026-03-01")
	if err == nil {
		t.Fatal("INSERT() error = nil, want database check constraint violation")
	}

	var databaseError *pgconn.PgError
	if !errors.As(err, &databaseError) {
		t.Fatalf("INSERT() error = %v, want PostgreSQL error", err)
	}
	if databaseError.Code != "23514" {
		t.Errorf("PostgreSQL error code = %q, want 23514", databaseError.Code)
	}
}

func newPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("projects_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		if dockerUnavailable(err) {
			t.Skipf("skipping PostgreSQL integration test because Docker is unavailable: %v", err)
		}
		t.Fatalf("start PostgreSQL test container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate PostgreSQL test container: %v", err)
		}
	})

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get PostgreSQL test connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		t.Fatalf("connect to PostgreSQL test container: %v", err)
	}
	t.Cleanup(pool.Close)
	waitForPostgres(t, pool)
	return pool
}

func waitForPostgres(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		lastErr = pool.Ping(ctx)
		cancel()
		if lastErr == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("wait for PostgreSQL test container: %v", lastErr)
}

func applyProjectsMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migration, err := os.ReadFile(filepath.Join(moduleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", "000001_create_projects.up.sql"))
	if err != nil {
		t.Fatalf("read projects migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply projects migration: %v", err)
	}
}

func applyStoriesMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migration, err := os.ReadFile(filepath.Join(moduleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", "000002_create_stories.up.sql"))
	if err != nil {
		t.Fatalf("read stories migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply stories migration: %v", err)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test package directory: %v", err)
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

func dockerUnavailable(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "permission denied while trying to connect to the docker api") ||
		strings.Contains(message, "cannot connect to the docker daemon") ||
		strings.Contains(message, "docker is not running")
}
