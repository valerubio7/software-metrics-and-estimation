package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
	taskpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/task/infrastructure/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/tests/integration/testpostgres"
)

const (
	taskProjectAID       = "82d38423-f02d-4259-9e35-4a291585bb1e"
	taskProjectBID       = "0b5cbb21-6a3e-4d4b-9b56-7d2f5d5f1c11"
	taskSprintS1ID       = "b2a6a455-06e2-41ee-b011-7462c71375a0"
	taskSprintS2ID       = "7256b0bb-835c-4166-9600-3d774e919477"
	taskStoryH1ID        = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	taskStoryH2ID        = "331a7ff5-12eb-4b2d-83fd-b51fd77e413d"
	taskProjectBSprintID = "ec44c1d7-5572-47d9-ad01-4fa0d79d8f00"
	taskProjectBStoryID  = "c7a1d2e3-4b5f-4c6d-8e7f-9a0b1c2d3e4f"
	missingTaskID        = "5f2d7c1e-8b4a-4a53-9c1d-3e6b7a9d0f12"
)

// taskFixture seeds project A with Sprint S1 (H1 assigned) and S2 (unassigned), H2
// unassigned, plus an unrelated project B with its own sprint and story.
func taskFixture(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := taskDatabase(t)
	insertProject(t, pool, taskProjectAID)
	insertProject(t, pool, taskProjectBID)
	insertSprint(t, pool, taskSprintS1ID, taskProjectAID)
	insertSprint(t, pool, taskSprintS2ID, taskProjectAID)
	insertStory(t, pool, taskStoryH1ID, taskProjectAID)
	insertStory(t, pool, taskStoryH2ID, taskProjectAID)
	assignStory(t, pool, taskSprintS1ID, taskStoryH1ID, taskProjectAID)
	insertSprint(t, pool, taskProjectBSprintID, taskProjectBID)
	insertStory(t, pool, taskProjectBStoryID, taskProjectBID)
	return pool
}

func TestTaskSchemaRequiresMigrationNine(t *testing.T) {
	pool := taskDatabase(t)
	var exists bool
	err := pool.QueryRow(context.Background(), "SELECT to_regclass('tasks') IS NOT NULL").Scan(&exists)
	if err != nil {
		t.Fatalf("inspect tasks table: %v", err)
	}
	if !exists {
		t.Fatal("tasks table does not exist after migration 000009")
	}
	for _, column := range []string{"id", "project_id", "sprint_id", "story_id", "title", "estimated_hours", "seq", "created_at"} {
		var present bool
		err := pool.QueryRow(context.Background(), `
			SELECT EXISTS (SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'tasks' AND column_name = $1)
		`, column).Scan(&present)
		if err != nil || !present {
			t.Errorf("tasks.%s missing (err=%v)", column, err)
		}
	}
}

func TestCreateTasksPersistsBatchLinkedToStorySprintAndProject(t *testing.T) {
	pool := taskFixture(t)
	repo := taskpostgres.NewPostgresTaskRepository(pool)
	estimate := 4.5
	tasks := []domain.Task{
		{ID: "9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10", Title: "Primera", EstimatedHours: &estimate},
		{ID: "1d6b5c58-1f87-4a5e-bb0f-0b2f8e4a7c21", Title: "Segunda sin estimar"},
	}

	err := repo.CreateForSprintStory(context.Background(), taskProjectAID, taskSprintS1ID, taskStoryH1ID, tasks)
	if err != nil {
		t.Fatalf("CreateForSprintStory() error = %v", err)
	}

	rows, err := pool.Query(context.Background(), `
		SELECT id::text, project_id::text, sprint_id::text, story_id::text, title, estimated_hours::text, seq, created_at IS NOT NULL
		FROM tasks ORDER BY seq
	`)
	if err != nil {
		t.Fatalf("read stored tasks: %v", err)
	}
	defer rows.Close()
	var sequences []int64
	count := 0
	for rows.Next() {
		var id, projectID, sprintID, storyID, title string
		var hours *string
		var seq int64
		var createdAt bool
		if err := rows.Scan(&id, &projectID, &sprintID, &storyID, &title, &hours, &seq, &createdAt); err != nil {
			t.Fatalf("scan stored task: %v", err)
		}
		if projectID != taskProjectAID || sprintID != taskSprintS1ID || storyID != taskStoryH1ID {
			t.Errorf("stored task association = (%s,%s,%s), want (%s,%s,%s)", projectID, sprintID, storyID, taskProjectAID, taskSprintS1ID, taskStoryH1ID)
		}
		if !createdAt {
			t.Error("created_at is NULL")
		}
		if id == tasks[0].ID && (hours == nil || *hours != "4.50") {
			t.Errorf("first task estimated_hours = %v, want 4.50", hours)
		}
		if id == tasks[1].ID && hours != nil {
			t.Errorf("second task estimated_hours = %v, want NULL", *hours)
		}
		sequences = append(sequences, seq)
		count++
	}
	if count != 2 {
		t.Fatalf("stored task count = %d, want 2", count)
	}
	if len(sequences) == 2 && sequences[0] >= sequences[1] {
		t.Errorf("seq order = %v, want increasing in batch order", sequences)
	}
}

func TestCreateTasksRejectsMissingOrForeignResourcesWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name      string
		projectID string
		sprintID  string
		storyID   string
		want      error
	}{
		{name: "missing project", projectID: missingTaskID, sprintID: taskSprintS1ID, storyID: taskStoryH1ID, want: application.ErrProjectNotFound},
		{name: "missing sprint", projectID: taskProjectAID, sprintID: missingTaskID, storyID: taskStoryH1ID, want: application.ErrSprintNotFound},
		{name: "sprint of another project", projectID: taskProjectAID, sprintID: taskProjectBSprintID, storyID: taskStoryH1ID, want: application.ErrSprintNotFound},
		{name: "missing story", projectID: taskProjectAID, sprintID: taskSprintS1ID, storyID: missingTaskID, want: application.ErrStoryNotFound},
		{name: "story of another project", projectID: taskProjectAID, sprintID: taskSprintS1ID, storyID: taskProjectBStoryID, want: application.ErrStoryNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := taskFixture(t)
			repo := taskpostgres.NewPostgresTaskRepository(pool)

			err := repo.CreateForSprintStory(context.Background(), tc.projectID, tc.sprintID, tc.storyID, []domain.Task{{ID: "9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10", Title: "Tarea"}})

			if !errors.Is(err, tc.want) {
				t.Errorf("CreateForSprintStory() error = %v, want %v", err, tc.want)
			}
			if got := taskCount(t, pool); got != 0 {
				t.Errorf("task count = %d, want 0", got)
			}
		})
	}
}

func TestCreateTasksRejectsStoryNotAssignedToSprint(t *testing.T) {
	for _, tc := range []struct {
		name     string
		sprintID string
		storyID  string
	}{
		{name: "story never assigned", sprintID: taskSprintS1ID, storyID: taskStoryH2ID},
		{name: "story assigned to a different sprint", sprintID: taskSprintS2ID, storyID: taskStoryH1ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := taskFixture(t)
			repo := taskpostgres.NewPostgresTaskRepository(pool)

			err := repo.CreateForSprintStory(context.Background(), taskProjectAID, tc.sprintID, tc.storyID, []domain.Task{{ID: "9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10", Title: "Tarea"}})

			if !errors.Is(err, application.ErrStoryNotInSprint) {
				t.Errorf("CreateForSprintStory() error = %v, want ErrStoryNotInSprint", err)
			}
			if got := taskCount(t, pool); got != 0 {
				t.Errorf("task count = %d, want 0", got)
			}
		})
	}
}

func TestCreateTasksAllowsClosedSprint(t *testing.T) {
	pool := taskFixture(t)
	if _, err := pool.Exec(context.Background(), "UPDATE sprints SET is_closed = true WHERE id = $1", taskSprintS1ID); err != nil {
		t.Fatalf("close sprint: %v", err)
	}
	repo := taskpostgres.NewPostgresTaskRepository(pool)

	err := repo.CreateForSprintStory(context.Background(), taskProjectAID, taskSprintS1ID, taskStoryH1ID, []domain.Task{{ID: "9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10", Title: "Tarea"}})

	if err != nil {
		t.Fatalf("CreateForSprintStory(closed sprint) error = %v, want nil", err)
	}
	if got := taskCount(t, pool); got != 1 {
		t.Errorf("task count = %d, want 1", got)
	}
}

func TestCreateTasksRollsBackWhenAnInsertFails(t *testing.T) {
	pool := taskFixture(t)
	if _, err := pool.Exec(context.Background(), `
		CREATE FUNCTION reject_second_task() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN IF NEW.title = 'Segunda' THEN RAISE EXCEPTION 'forced write failure'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER reject_task BEFORE INSERT ON tasks FOR EACH ROW EXECUTE FUNCTION reject_second_task();
	`); err != nil {
		t.Fatalf("install failing trigger: %v", err)
	}
	repo := taskpostgres.NewPostgresTaskRepository(pool)

	err := repo.CreateForSprintStory(context.Background(), taskProjectAID, taskSprintS1ID, taskStoryH1ID, []domain.Task{
		{ID: "9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10", Title: "Primera"},
		{ID: "1d6b5c58-1f87-4a5e-bb0f-0b2f8e4a7c21", Title: "Segunda"},
	})

	if err == nil || errors.Is(err, application.ErrStoryNotInSprint) {
		t.Fatalf("forced insert failure misclassified: %v", err)
	}
	if got := taskCount(t, pool); got != 0 {
		t.Errorf("task count = %d, want 0 (rollback)", got)
	}
}

func TestTasksTableEnforcesConstraints(t *testing.T) {
	pool := taskFixture(t)

	t.Run("sprint-story pair not assigned", func(t *testing.T) {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO tasks (id, project_id, sprint_id, story_id, title) VALUES
			('9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10', $1, $2, $3, 'Tarea')
		`, taskProjectAID, taskSprintS1ID, taskStoryH2ID)
		assertDatabaseError(t, err, "23503", "tasks_sprint_story_fkey")
	})

	t.Run("assigned pair but project_id from another project", func(t *testing.T) {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO tasks (id, project_id, sprint_id, story_id, title) VALUES
			('1d6b5c58-1f87-4a5e-bb0f-0b2f8e4a7c21', $1, $2, $3, 'Tarea')
		`, taskProjectBID, taskSprintS1ID, taskStoryH1ID)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23503" ||
			(pgErr.ConstraintName != "tasks_sprint_project_fkey" && pgErr.ConstraintName != "tasks_story_project_fkey") {
			t.Errorf("error = %v, want 23503 on tasks_sprint_project_fkey or tasks_story_project_fkey", err)
		}
	})

	t.Run("zero estimate", func(t *testing.T) {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours) VALUES
			('9eaa6c2a-8b8a-4e5f-9b0a-7e8c7e6c7a1a', $1, $2, $3, 'Tarea', 0)
		`, taskProjectAID, taskSprintS1ID, taskStoryH1ID)
		assertDatabaseError(t, err, "23514", "tasks_estimated_hours_positive")
	})

	t.Run("estimate above NUMERIC(7,2)", func(t *testing.T) {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours) VALUES
			('c1b2a3d4-5e6f-4a7b-8c9d-0e1f2a3b4c5d', $1, $2, $3, 'Tarea', 100000)
		`, taskProjectAID, taskSprintS1ID, taskStoryH1ID)
		assertDatabaseError(t, err, "22003", "")
	})

	if got := taskCount(t, pool); got != 0 {
		t.Errorf("task count after rejected inserts = %d, want 0", got)
	}
}

func TestTaskBlocksUnassigningItsStory(t *testing.T) {
	pool := taskFixture(t)
	repo := taskpostgres.NewPostgresTaskRepository(pool)
	if err := repo.CreateForSprintStory(context.Background(), taskProjectAID, taskSprintS1ID, taskStoryH1ID, []domain.Task{{ID: "9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10", Title: "Tarea"}}); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	_, err := pool.Exec(context.Background(), "DELETE FROM sprint_stories WHERE sprint_id = $1 AND story_id = $2", taskSprintS1ID, taskStoryH1ID)

	assertDatabaseError(t, err, "23503", "tasks_sprint_story_fkey")
}

func TestTasksMigrationCanBeReversedAndReapplied(t *testing.T) {
	pool := taskFixture(t)
	applyTaskMigration(t, pool, "000009_create_tasks.down.sql")

	var tasksExists, sprintStoriesExists bool
	if err := pool.QueryRow(context.Background(), "SELECT to_regclass('tasks') IS NULL, to_regclass('sprint_stories') IS NOT NULL").Scan(&tasksExists, &sprintStoriesExists); err != nil {
		t.Fatalf("inspect schema after down: %v", err)
	}
	if !tasksExists || !sprintStoriesExists {
		t.Fatalf("after down: tasks gone=%v, sprint_stories intact=%v", tasksExists, sprintStoriesExists)
	}

	applyTaskMigration(t, pool, "000009_create_tasks.up.sql")

	var recreated bool
	if err := pool.QueryRow(context.Background(), "SELECT to_regclass('tasks') IS NOT NULL").Scan(&recreated); err != nil {
		t.Fatalf("inspect schema after reapplied up: %v", err)
	}
	if !recreated {
		t.Fatal("tasks table missing after reapplying migration 000009")
	}
}

func insertSprint(t *testing.T, pool *pgxpool.Pool, id, projectID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Delivery')", id, projectID)
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
}

func insertStory(t *testing.T, pool *pgxpool.Pool, id, projectID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO stories (id, project_id, title, description, priority, status, acceptance_criteria)
		VALUES ($1, $2, 'Historia', 'Detalle', 'media', 'pendiente', ARRAY['Listo'])
	`, id, projectID)
	if err != nil {
		t.Fatalf("insert story: %v", err)
	}
}

func assignStory(t *testing.T, pool *pgxpool.Pool, sprintID, storyID, projectID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "INSERT INTO sprint_stories (sprint_id, story_id, project_id) VALUES ($1, $2, $3)", sprintID, storyID, projectID)
	if err != nil {
		t.Fatalf("assign story to sprint: %v", err)
	}
}

func taskCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM tasks").Scan(&count); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	return count
}

func assertDatabaseError(t *testing.T, err error, code, constraint string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error = %v, want PostgreSQL %s on %s", err, code, constraint)
	}
	if pgErr.Code != code || (constraint != "" && pgErr.ConstraintName != constraint) {
		t.Errorf("database error = (%s, %s), want (%s, %s)", pgErr.Code, pgErr.ConstraintName, code, constraint)
	}
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

func taskDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	if dsn := os.Getenv("TASK_TEST_DATABASE_URL"); dsn != "" {
		return taskDatabaseFromDSN(t, dsn)
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("task_test"),
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
	waitForTaskDatabaseReady(t, ctx, pool)
	for _, name := range canonicalTaskMigrations() {
		applyTaskMigration(t, pool, name)
	}
	return pool
}

// waitForTaskDatabaseReady polls with Ping until the container accepts connections.
// The container's "ready" log line fires before PostgreSQL finishes starting up;
// connecting immediately can fail with an EOF on the wire.
func waitForTaskDatabaseReady(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		pingCtx, cancel := context.WithTimeout(ctx, time.Second)
		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("wait for PostgreSQL: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func taskDatabaseFromDSN(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool := testpostgres.OpenIsolated(t, dsn, "TASK_TEST_DATABASE_URL", "task_test")
	for _, name := range canonicalTaskMigrations() {
		applyTaskMigration(t, pool, name)
	}
	return pool
}

func canonicalTaskMigrations() []string {
	return []string{
		"000001_create_projects.up.sql",
		"000002_create_stories.up.sql",
		"000003_add_story_estimated_hours.up.sql",
		"000004_add_story_creation_sequence.up.sql",
		"000005_reconcile_story_hours_and_sprints.up.sql",
		"000006_create_project_members.up.sql",
		"000007_create_sprint_stories.up.sql",
		"000008_add_sprint_closed.up.sql",
		"000009_create_tasks.up.sql",
	}
}

// applyTaskMigration executes one migration file from the shared migrations directory.
func applyTaskMigration(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	migration, err := os.ReadFile(filepath.Join(taskModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", name))
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply migration %s: %v", name, err)
	}
}

func taskModuleRoot(t *testing.T) string {
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
