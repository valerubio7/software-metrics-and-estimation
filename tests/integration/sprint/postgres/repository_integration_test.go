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
	"github.com/valerubio7/software-metrics-and-estimation/tests/integration/testpostgres"
)

const (
	projectID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	sprintID  = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
)

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

func TestCloseSprintPersistsClosureAndPreservesAssociations(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	_, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Preserve history')", sprintID, projectID)
	if err != nil {
		t.Fatalf("seed sprint: %v", err)
	}
	storyID := "a25b26b4-26bf-482e-a318-74e6cb845462"
	_, err = pool.Exec(context.Background(), `
		INSERT INTO stories (id, project_id, title, description, priority, status, acceptance_criteria)
		VALUES ($1, $2, 'Story', 'Description', 'alta', 'completada', ARRAY['criterion'])
	`, storyID, projectID)
	if err != nil {
		t.Fatalf("seed story: %v", err)
	}
	_, err = pool.Exec(context.Background(), "INSERT INTO sprint_stories (sprint_id, story_id, project_id, completed_at) VALUES ($1, $2, $3, '2026-10-11T10:00:00Z')", sprintID, storyID, projectID)
	if err != nil {
		t.Fatalf("seed sprint association: %v", err)
	}
	taskID := "c631b683-b585-4df3-b6a3-10ed2fd11d9e"
	_, err = pool.Exec(context.Background(), `
		INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours)
		VALUES ($1, $2, $3, $4, 'Preserved task', 2.5)
	`, taskID, projectID, sprintID, storyID)
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	repository := sprintpostgres.NewPostgresSprintRepository(pool)
	if err := repository.CloseSprint(context.Background(), projectID, sprintID); err != nil {
		t.Fatalf("CloseSprint() error = %v", err)
	}
	var closed bool
	var associationCount, taskCount int
	var completedAt *time.Time
	var storyStatus string
	var taskTitle string
	var estimatedHours float64
	if err := pool.QueryRow(context.Background(), "SELECT is_closed FROM sprints WHERE id = $1", sprintID).Scan(&closed); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*), max(completed_at) FROM sprint_stories WHERE sprint_id = $1", sprintID).Scan(&associationCount, &completedAt); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT status FROM stories WHERE id = $1", storyID).Scan(&storyStatus); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*), max(title), max(estimated_hours) FROM tasks WHERE sprint_id = $1", sprintID).Scan(&taskCount, &taskTitle, &estimatedHours); err != nil {
		t.Fatal(err)
	}
	if !closed || associationCount != 1 || completedAt == nil || !completedAt.Equal(time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)) || storyStatus != "completada" || taskCount != 1 || taskTitle != "Preserved task" || estimatedHours != 2.5 {
		t.Fatalf("closed/history = %v/%d/%v/%s/%d/%q/%.2f, want closed and unchanged completion, story, task and effort", closed, associationCount, completedAt, storyStatus, taskCount, taskTitle, estimatedHours)
	}
	if err := repository.CloseSprint(context.Background(), projectID, sprintID); !errors.Is(err, application.ErrSprintClosed) {
		t.Errorf("repeated CloseSprint() error = %v, want ErrSprintClosed", err)
	}
	if err := repository.CloseSprint(context.Background(), "9b6847be-c44c-47e8-a91d-69aa2874a80f", sprintID); !errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("CloseSprint(missing project) error = %v, want ErrProjectNotFound", err)
	}
	if err := repository.CloseSprint(context.Background(), projectID, "9b6847be-c44c-47e8-a91d-69aa2874a80f"); !errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("CloseSprint(missing sprint) error = %v, want ErrSprintNotFound", err)
	}
	otherProjectID := "8756a2df-783d-4070-a236-b4284356ed50"
	otherSprintID := "4991929f-4ef2-4412-9365-04235c3c1dbf"
	if _, err := pool.Exec(context.Background(), `INSERT INTO projects (id, name, start_date, planned_finish_date) VALUES ($1, 'Other', '2026-03-01', '2026-06-30')`, otherProjectID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Other sprint')", otherSprintID, otherProjectID); err != nil {
		t.Fatal(err)
	}
	if err := repository.CloseSprint(context.Background(), projectID, otherSprintID); !errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("CloseSprint(other project) error = %v, want ErrSprintNotFound", err)
	}
	var otherClosed bool
	if err := pool.QueryRow(context.Background(), "SELECT is_closed FROM sprints WHERE id = $1", otherSprintID).Scan(&otherClosed); err != nil {
		t.Fatal(err)
	}
	if otherClosed {
		t.Errorf("cross-project rejection changed sprint state: is_closed=%v", otherClosed)
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

const sprintSeqConstraint = "sprints_project_id_seq_key"

func TestSprintsSeqIsAnAlwaysIdentityNotNullColumn(t *testing.T) {
	pool := sprintDatabase(t)

	var dataType, nullable, isIdentity, generation string
	err := pool.QueryRow(context.Background(), `
		SELECT data_type, is_nullable, is_identity, identity_generation FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'sprints' AND column_name = 'seq'
	`).Scan(&dataType, &nullable, &isIdentity, &generation)
	if err != nil {
		t.Fatalf("inspect sprints.seq: %v", err)
	}
	if dataType != "bigint" || nullable != "NO" || isIdentity != "YES" || generation != "ALWAYS" {
		t.Errorf("sprints.seq = (%s, nullable %s, identity %s, %s), want (bigint, nullable NO, identity YES, ALWAYS)",
			dataType, nullable, isIdentity, generation)
	}
}

func TestSprintsSeqIsUniquePerProject(t *testing.T) {
	pool := sprintDatabase(t)
	var definition string
	err := pool.QueryRow(context.Background(), `
		SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'sprints'::regclass AND conname = $1
	`, sprintSeqConstraint).Scan(&definition)
	if err != nil {
		t.Fatalf("named seq constraint missing: %v", err)
	}
	if definition != "UNIQUE (project_id, seq)" {
		t.Errorf("%s = %q, want UNIQUE (project_id, seq)", sprintSeqConstraint, definition)
	}

	insertProject(t, pool)
	repository := sprintpostgres.NewPostgresSprintRepository(pool)
	if err := repository.Create(context.Background(), domain.Sprint{ID: sprintID, ProjectID: projectID, SprintGoal: "First"}); err != nil {
		t.Fatalf("seed sprint: %v", err)
	}
	// Only OVERRIDING SYSTEM VALUE can repeat a seq, which is how the constraint is exercised.
	_, err = pool.Exec(context.Background(), `
		INSERT INTO sprints (id, project_id, sprint_goal, seq) OVERRIDING SYSTEM VALUE
		SELECT $1::uuid, project_id, 'Duplicate seq', seq FROM sprints WHERE id = $2
	`, "4991929f-4ef2-4412-9365-04235c3c1dbf", sprintID)
	assertDatabaseError(t, err, "23505", sprintSeqConstraint)
	if got := sprintCount(t, pool); got != 1 {
		t.Errorf("sprint count after rejected duplicate seq = %d, want 1", got)
	}
}

func TestSprintsSeqRejectsExplicitValuesWithoutOverriding(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)

	_, err := pool.Exec(context.Background(), `
		INSERT INTO sprints (id, project_id, sprint_goal, seq) VALUES ($1, $2, 'Explicit seq', 99)
	`, sprintID, projectID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "428C9" {
		t.Fatalf("explicit seq error = %v, want PostgreSQL 428C9", err)
	}
	if got := sprintCount(t, pool); got != 0 {
		t.Errorf("sprint count after rejected explicit seq = %d, want 0", got)
	}
}

func TestSprintCreateAssignsIncreasingSeq(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	repository := sprintpostgres.NewPostgresSprintRepository(pool)
	ids := []string{sprintID, "4991929f-4ef2-4412-9365-04235c3c1dbf"}
	for index, id := range ids {
		sprint := domain.Sprint{ID: id, ProjectID: projectID, SprintGoal: "Sprint " + string(rune('A'+index))}
		if err := repository.Create(context.Background(), sprint); err != nil {
			t.Fatalf("Create(%s) error = %v", id, err)
		}
	}

	var first, second int64
	if err := pool.QueryRow(context.Background(), "SELECT seq FROM sprints WHERE id = $1", ids[0]).Scan(&first); err != nil {
		t.Fatalf("read first seq: %v", err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT seq FROM sprints WHERE id = $1", ids[1]).Scan(&second); err != nil {
		t.Fatalf("read second seq: %v", err)
	}
	if second <= first {
		t.Errorf("seq of second sprint = %d, want greater than first sprint seq %d", second, first)
	}
}

func TestSprintCreationSequenceMigrationCanBeReversedAndReapplied(t *testing.T) {
	pool := sprintDatabase(t)
	ctx := context.Background()
	migrations := sprintMigrationsDir(t)
	seedClosedSprintHistory(t, pool)
	before := sprintBusinessSnapshot(t, pool)

	applySprintMigration(t, pool, migrations, "000011_add_sprint_creation_sequence.down.sql")
	var column, constraint int
	err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'sprints' AND column_name = 'seq'),
		       (SELECT count(*) FROM pg_constraint WHERE conrelid = 'sprints'::regclass AND conname = $1)
	`, sprintSeqConstraint).Scan(&column, &constraint)
	if err != nil {
		t.Fatalf("inspect schema after down: %v", err)
	}
	if column != 0 || constraint != 0 {
		t.Fatalf("after down: seq column/constraint = %d/%d, want 0/0", column, constraint)
	}
	if after := sprintBusinessSnapshot(t, pool); after != before {
		t.Fatalf("business data changed by down:\nbefore:\n%s\nafter:\n%s", before, after)
	}

	applySprintMigration(t, pool, migrations, "000011_add_sprint_creation_sequence.up.sql")
	var sprints, distinct, withSeq int
	err = pool.QueryRow(ctx, `
		SELECT count(*), count(DISTINCT (project_id, seq)), count(seq) FROM sprints
	`).Scan(&sprints, &distinct, &withSeq)
	if err != nil {
		t.Fatalf("inspect seq after reapply: %v", err)
	}
	if sprints != 3 || distinct != 3 || withSeq != 3 {
		t.Errorf("after up: sprints/distinct seq/non-null seq = %d/%d/%d, want 3/3/3", sprints, distinct, withSeq)
	}
	if after := sprintBusinessSnapshot(t, pool); after != before {
		t.Errorf("business data changed by reapplying up:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// seedClosedSprintHistory seeds one closed and two open sprints with an association, a completion and a
// task, so that the migration round trip covers business data.
func seedClosedSprintHistory(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	insertProject(t, pool)
	storyID := "a25b26b4-26bf-482e-a318-74e6cb845462"
	statements := []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO sprints (id, project_id, sprint_goal, is_closed) VALUES ($1, $2, 'Closed sprint', true)", []any{sprintID, projectID}},
		{"INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Open sprint')", []any{"4991929f-4ef2-4412-9365-04235c3c1dbf", projectID}},
		{"INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Second open sprint')", []any{"0b3c7d52-1a43-4f5e-9d4c-6a2f7a1e9b10", projectID}},
		{`INSERT INTO stories (id, project_id, title, description, priority, status, acceptance_criteria)
		  VALUES ($1, $2, 'Story', 'Description', 'alta', 'completada', ARRAY['criterion'])`, []any{storyID, projectID}},
		{"INSERT INTO sprint_stories (sprint_id, story_id, project_id, completed_at) VALUES ($1, $2, $3, '2026-10-11T10:00:00Z')", []any{sprintID, storyID, projectID}},
		{"INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours) VALUES ($1, $2, $3, $4, 'Preserved task', 2.5)",
			[]any{"c631b683-b585-4df3-b6a3-10ed2fd11d9e", projectID, sprintID, storyID}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(context.Background(), statement.sql, statement.args...); err != nil {
			t.Fatalf("seed %q: %v", statement.sql, err)
		}
	}
}

// sprintBusinessSnapshot renders the business columns of the sprint tables, excluding the seq column
// that the migration adds and removes.
func sprintBusinessSnapshot(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var snapshot string
	err := pool.QueryRow(context.Background(), `
		SELECT concat_ws(E'\n',
			(SELECT coalesce(string_agg((to_jsonb(s) - 'seq')::text, E'\n' ORDER BY id), '') FROM sprints s),
			(SELECT coalesce(string_agg(to_jsonb(ss)::text, E'\n' ORDER BY sprint_id, story_id), '') FROM sprint_stories ss),
			(SELECT coalesce(string_agg(to_jsonb(st)::text, E'\n' ORDER BY id), '') FROM stories st),
			(SELECT coalesce(string_agg(to_jsonb(tk)::text, E'\n' ORDER BY id), '') FROM tasks tk))
	`).Scan(&snapshot)
	if err != nil {
		t.Fatalf("snapshot sprint business data: %v", err)
	}
	return snapshot
}

func sprintMigrationsDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(sprintModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations")
}

func sprintDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	if dsn := os.Getenv("SPRINT_TEST_DATABASE_URL"); dsn != "" {
		return sprintDatabaseFromDSN(t, dsn)
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
	root := sprintModuleRoot(t)
	migrations := filepath.Join(root, "internal", "project", "infrastructure", "postgres", "migrations")
	for _, name := range sprintMigrationNames {
		applySprintMigration(t, pool, migrations, name)
	}
	return pool
}

var sprintMigrationNames = []string{
	"000001_create_projects.up.sql", "000002_create_stories.up.sql",
	"000003_add_story_estimated_hours.up.sql", "000004_add_story_creation_sequence.up.sql",
	"000005_reconcile_story_hours_and_sprints.up.sql", "000006_create_project_members.up.sql",
	"000007_create_sprint_stories.up.sql", "000008_add_sprint_closed.up.sql",
	"000009_create_tasks.up.sql", "000010_add_sprint_story_completion.up.sql",
	"000011_add_sprint_creation_sequence.up.sql",
}

func sprintDatabaseFromDSN(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool := testpostgres.OpenIsolated(t, dsn, "SPRINT_TEST_DATABASE_URL", "sprint_test")
	root := sprintModuleRoot(t)
	migrations := filepath.Join(root, "internal", "project", "infrastructure", "postgres", "migrations")
	for _, name := range sprintMigrationNames {
		applySprintMigration(t, pool, migrations, name)
	}
	return pool
}

func applySprintMigration(t *testing.T, pool *pgxpool.Pool, directory, name string) {
	t.Helper()
	migration, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply migration %s: %v", name, err)
	}
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
