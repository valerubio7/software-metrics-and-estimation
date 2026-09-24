package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
)

const (
	projectID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	storyID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
)

func TestStoryRepositoryStoresLinkedUnestimatedStory(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	story := validStory(projectID)

	err := storypostgres.NewPostgresStoryRepository(pool).Create(context.Background(), story)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var id, linkedProject, title, description, priority, status string
	var pointsAreNull bool
	var criteria []string
	err = pool.QueryRow(context.Background(), `
		SELECT id::text, project_id::text, title, description, priority, status,
		       story_points IS NULL, acceptance_criteria
		FROM stories WHERE id = $1
	`, story.ID).Scan(&id, &linkedProject, &title, &description, &priority, &status, &pointsAreNull, &criteria)
	if err != nil {
		t.Fatalf("read stored story: %v", err)
	}
	if id != story.ID || linkedProject != projectID || title != story.Title ||
		description != story.Description || priority != story.Priority || status != "pendiente" {
		t.Errorf("stored story = (%q, %q, %q, %q, %q, %q), want story fields and pendiente",
			id, linkedProject, title, description, priority, status)
	}
	if !pointsAreNull {
		t.Error("story_points is not SQL NULL")
	}
	if !reflect.DeepEqual(criteria, story.AcceptanceCriteria) {
		t.Errorf("acceptance_criteria = %#v, want %#v", criteria, story.AcceptanceCriteria)
	}
	if got := storyCount(t, pool); got != 1 {
		t.Errorf("story count = %d, want 1", got)
	}
}

func TestStoryRepositoryRejectsMissingProjectWithNamedForeignKey(t *testing.T) {
	pool := storyDatabase(t)
	story := validStory(projectID)

	var constraintDefinition string
	err := pool.QueryRow(context.Background(), `
		SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'stories'::regclass AND conname = 'stories_project_id_fkey'
	`).Scan(&constraintDefinition)
	if err != nil {
		t.Fatalf("named FK missing: %v", err)
	}
	if !strings.Contains(constraintDefinition, "REFERENCES projects(id) ON DELETE RESTRICT") {
		t.Errorf("named FK = %q, want projects(id) ON DELETE RESTRICT", constraintDefinition)
	}

	err = storypostgres.NewPostgresStoryRepository(pool).Create(context.Background(), story)
	if !errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("Create(missing project) error = %v, want ErrProjectNotFound", err)
	}
	if got := storyCount(t, pool); got != 0 {
		t.Errorf("story count after missing project = %d, want 0", got)
	}
}

func TestStoryRepositoryDoesNotMislabelUnexpectedDatabaseErrors(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	repository := storypostgres.NewPostgresStoryRepository(pool)
	story := validStory(projectID)
	if err := repository.Create(context.Background(), story); err != nil {
		t.Fatalf("seed story: %v", err)
	}

	// A second write with the same ID violates the PK, not the project FK.
	err := repository.Create(context.Background(), story)
	assertDatabaseError(t, err, "23505", "stories_pkey")
	if errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("duplicate ID mislabeled as missing project: %v", err)
	}

	// A different 23503 must not be translated just because its SQLSTATE matches.
	_, err = pool.Exec(context.Background(), `
		CREATE FUNCTION reject_story_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'other relationship failed' USING ERRCODE = '23503', CONSTRAINT = 'other_project_fkey';
		END $$;
		CREATE TRIGGER reject_story BEFORE INSERT ON stories
		FOR EACH ROW EXECUTE FUNCTION reject_story_insert();
	`)
	if err != nil {
		t.Fatalf("install unrelated FK failure trigger: %v", err)
	}
	story.ID = "331a7ff5-12eb-4b2d-83fd-b51fd77e413d"
	err = repository.Create(context.Background(), story)
	assertDatabaseError(t, err, "23503", "other_project_fkey")
	if errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("unrelated 23503 mislabeled as missing project: %v", err)
	}
	if got := storyCount(t, pool); got != 1 {
		t.Errorf("story count after failed inserts = %d, want 1", got)
	}
}

func TestStoriesForeignKeyPreventsOrphansInBothWriteOrders(t *testing.T) {
	for _, tc := range []struct {
		name         string
		insertFirst  bool
		wantProjects int
		wantStories  int
	}{
		{name: "insert before delete", insertFirst: true, wantProjects: 1, wantStories: 1},
		{name: "delete before insert", wantProjects: 0, wantStories: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := storyDatabase(t)
			insertProject(t, pool, projectID)
			repository := storypostgres.NewPostgresStoryRepository(pool)
			story := validStory(projectID)
			if tc.insertFirst {
				if err := repository.Create(context.Background(), story); err != nil {
					t.Fatalf("insert before delete: %v", err)
				}
			}
			_, err := pool.Exec(context.Background(), "DELETE FROM projects WHERE id = $1", projectID)
			if tc.insertFirst {
				assertDatabaseError(t, err, "23503", "stories_project_id_fkey")
			} else if err != nil {
				t.Fatalf("delete before insert: %v", err)
			}
			if !tc.insertFirst {
				err = repository.Create(context.Background(), story)
				if !errors.Is(err, application.ErrProjectNotFound) {
					t.Errorf("insert after delete error = %v, want ErrProjectNotFound", err)
				}
			}
			if got := storyCount(t, pool); got != tc.wantStories {
				t.Errorf("story count = %d, want %d", got, tc.wantStories)
			}
			var projects int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM projects").Scan(&projects); err != nil {
				t.Fatalf("count projects: %v", err)
			}
			if projects != tc.wantProjects {
				t.Errorf("project count = %d, want %d", projects, tc.wantProjects)
			}
			var orphans int
			if err := pool.QueryRow(context.Background(), `
				SELECT count(*) FROM stories s LEFT JOIN projects p ON p.id = s.project_id WHERE p.id IS NULL
			`).Scan(&orphans); err != nil {
				t.Fatalf("count orphans: %v", err)
			}
			if orphans != 0 {
				t.Errorf("orphan count = %d, want 0", orphans)
			}
		})
	}
}

// The first transaction remains open until PostgreSQL reports the competing
// statement waiting on its lock. This tests overlap, not just two serial orders.
func TestStoriesForeignKeyPreservesIntegrityDuringConcurrentInsertAndDelete(t *testing.T) {
	for _, insertWins := range []bool{true, false} {
		name := "delete commits first"
		if insertWins {
			name = "insert commits first"
		}
		t.Run(name, func(t *testing.T) {
			pool := storyDatabase(t)
			insertProject(t, pool, projectID)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatalf("begin first writer: %v", err)
			}
			defer tx.Rollback(context.Background())

			story := validStory(projectID)
			waitingQuery := "INSERT INTO stories"
			if insertWins {
				_, err = tx.Exec(ctx, `
					INSERT INTO stories (id, project_id, title, description, priority, status, story_points, acceptance_criteria)
					VALUES ($1, $2, $3, $4, $5, $6, NULL, $7)
				`, story.ID, story.ProjectID, story.Title, story.Description, story.Priority, story.Status, story.AcceptanceCriteria)
				waitingQuery = "DELETE FROM projects"
			} else {
				_, err = tx.Exec(ctx, "DELETE FROM projects WHERE id = $1", projectID)
			}
			if err != nil {
				t.Fatalf("first uncommitted write: %v", err)
			}

			result := make(chan error, 1)
			go func() {
				if insertWins {
					_, err := pool.Exec(ctx, "DELETE FROM projects WHERE id = $1", projectID)
					result <- err
				} else {
					result <- storypostgres.NewPostgresStoryRepository(pool).Create(ctx, story)
				}
			}()
			waitForStoryWriteLock(t, ctx, pool, waitingQuery, result)
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("commit first writer: %v", err)
			}
			select {
			case err := <-result:
				if insertWins {
					assertDatabaseError(t, err, "23503", "stories_project_id_fkey")
				} else if !errors.Is(err, application.ErrProjectNotFound) {
					t.Errorf("concurrent insert error = %v, want ErrProjectNotFound", err)
				}
			case <-ctx.Done():
				t.Fatalf("waiting writer did not finish: %v", ctx.Err())
			}

			want := 0
			if insertWins {
				want = 1
			}
			if got := storyCount(t, pool); got != want {
				t.Errorf("story count = %d, want %d", got, want)
			}
			var projects, orphans int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM projects").Scan(&projects); err != nil {
				t.Fatalf("count projects: %v", err)
			}
			if err := pool.QueryRow(ctx, `
				SELECT count(*) FROM stories s LEFT JOIN projects p ON p.id = s.project_id WHERE p.id IS NULL
			`).Scan(&orphans); err != nil {
				t.Fatalf("count orphans: %v", err)
			}
			if projects != want || orphans != 0 {
				t.Errorf("projects/orphans = %d/%d, want %d/0", projects, orphans, want)
			}
		})
	}
}

func waitForStoryWriteLock(t *testing.T, ctx context.Context, pool *pgxpool.Pool, statement string, result <-chan error) {
	t.Helper()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE pid <> pg_backend_pid() AND wait_event_type = 'Lock'
				AND position($1 in query) > 0
			)
		`, statement).Scan(&blocked)
		if err != nil {
			t.Fatalf("inspect waiting writer: %v", err)
		}
		if blocked {
			return
		}
		select {
		case err := <-result:
			t.Fatalf("writer completed without observed lock wait: %v", err)
		case <-ctx.Done():
			t.Fatalf("writer never waited for first transaction: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func validStory(project string) domain.Story {
	return domain.Story{
		ID: storyID, ProjectID: project, Title: "  Registro  ", Description: "Detalle original",
		Priority: "media", Status: "pendiente", AcceptanceCriteria: []string{" Primero ", "Segundo"},
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

func storyCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM stories").Scan(&count); err != nil {
		t.Fatalf("count stories: %v", err)
	}
	return count
}

func insertProject(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO projects (id, name, start_date, planned_finish_date)
		VALUES ($1, 'Backlog', '2026-03-01', '2026-06-30')
	`, id)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
}

func storyDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stories_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL container (no silent Docker skip): %v", err)
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
	for _, name := range []string{"000001_create_projects.up.sql", "000002_create_stories.up.sql"} {
		migration, err := os.ReadFile(filepath.Join(storyModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
	return pool
}

func storyModuleRoot(t *testing.T) string {
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
