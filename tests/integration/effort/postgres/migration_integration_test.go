package postgres_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/tests/integration/testpostgres"
)

const (
	effortProjectAID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	effortSprintID   = "b2a6a455-06e2-41ee-b011-7462c71375a0"
	effortStoryID    = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	effortMemberID   = "7256b0bb-835c-4166-9600-3d774e919477"
	effortTaskID     = "9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10"
	effortRecordID   = "1d6b5c58-1f87-4a5e-bb0f-0b2f8e4a7c21"
	effortRecordID2  = "c7a1d2e3-4b5f-4c6d-8e7f-9a0b1c2d3e4f"
)

const (
	effortMigrationVersion    = "000012"
	lastPreEffortMigration    = 11
	effortStoryProjectFKey    = "effort_records_story_project_fkey"
	effortMemberProjectFKey   = "effort_records_member_project_fkey"
	effortTaskStoryProjFKey   = "effort_records_task_story_project_fkey"
	membersUniqueConstraint   = "project_members_id_project_id_key"
	tasksUniqueConstraint     = "tasks_id_story_id_project_id_key"
	effortHoursRangeCheck     = "effort_records_hours_range"
	effortDescriptionLenCheck = "effort_records_description_length"
)

func TestEffortMigrationCreatesSchema(t *testing.T) {
	pool := effortDatabase(t)

	assertEffortSchema(t, pool)
}

func TestEffortMigrationUpOverVersion11KeepsExistingData(t *testing.T) {
	pool := effortDatabaseAtVersion(t, lastPreEffortMigration)
	seedEffortParents(t, pool)
	if exists := relationExists(t, pool, "effort_records"); exists {
		t.Fatal("effort_records must not exist at schema version 11")
	}

	applyEffortMigration(t, pool, effortMigrationVersion, "up")

	assertEffortSchema(t, pool)
	assertParentsIntact(t, pool)
	if got := effortRecordCount(t, pool); got != 0 {
		t.Errorf("effort_records rows after up = %d, want 0", got)
	}
}

func TestEffortMigrationDownRemovesSchemaAndKeepsData(t *testing.T) {
	pool := effortDatabase(t)
	seedEffortParents(t, pool)
	insertEffortRecord(t, pool, effortRecordID, nil)

	applyEffortMigration(t, pool, effortMigrationVersion, "down")

	assertEffortSchemaAbsent(t, pool)
	assertParentsIntact(t, pool)
}

func TestEffortMigrationDownIsIdempotent(t *testing.T) {
	pool := effortDatabase(t)
	seedEffortParents(t, pool)

	applyEffortMigration(t, pool, effortMigrationVersion, "down")
	applyEffortMigration(t, pool, effortMigrationVersion, "down")

	assertEffortSchemaAbsent(t, pool)
	assertParentsIntact(t, pool)
}

func TestEffortMigrationCanBeReapplied(t *testing.T) {
	pool := effortDatabase(t)
	seedEffortParents(t, pool)
	applyEffortMigration(t, pool, effortMigrationVersion, "down")

	applyEffortMigration(t, pool, effortMigrationVersion, "up")

	assertEffortSchema(t, pool)
	assertParentsIntact(t, pool)
	taskID := effortTaskID
	insertEffortRecord(t, pool, effortRecordID, nil)
	insertEffortRecord(t, pool, effortRecordID2, &taskID)
	if got := effortRecordCount(t, pool); got != 2 {
		t.Errorf("effort_records rows after reapplied up = %d, want 2 (story-level with NULL task_id and task-level)", got)
	}
}

// assertEffortSchema verifies the full effort_records contract by reading the catalog.
func assertEffortSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	wantColumns := map[string]string{
		"id":          "uuid/NO",
		"project_id":  "uuid/NO",
		"story_id":    "uuid/NO",
		"task_id":     "uuid/YES",
		"member_id":   "uuid/NO",
		"work_date":   "date/NO",
		"description": "text/NO",
		"hours":       "numeric(5,2)/NO",
		"seq":         "bigint/NO",
		"created_at":  "timestamp with time zone/NO",
	}
	gotColumns := make(map[string]string)
	rows, err := pool.Query(ctx, `
		SELECT a.attname,
		       format_type(a.atttypid, a.atttypmod) || '/' || CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END
		FROM pg_attribute a
		WHERE a.attrelid = to_regclass('effort_records') AND a.attnum > 0 AND NOT a.attisdropped
	`)
	if err != nil {
		t.Fatalf("read effort_records columns: %v", err)
	}
	for rows.Next() {
		var name, shape string
		if err := rows.Scan(&name, &shape); err != nil {
			rows.Close()
			t.Fatalf("scan effort_records column: %v", err)
		}
		gotColumns[name] = shape
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate effort_records columns: %v", err)
	}
	if !reflect.DeepEqual(gotColumns, wantColumns) {
		t.Errorf("effort_records columns = %v, want %v", gotColumns, wantColumns)
	}

	// Foreign keys: composite, MATCH SIMPLE (confmatchtype 's') and ON DELETE RESTRICT (confdeltype 'r').
	wantForeignKeys := map[string]string{
		effortStoryProjectFKey:  "FOREIGN KEY (story_id, project_id) REFERENCES stories(id, project_id) ON DELETE RESTRICT/s/r",
		effortMemberProjectFKey: "FOREIGN KEY (member_id, project_id) REFERENCES project_members(id, project_id) ON DELETE RESTRICT/s/r",
		effortTaskStoryProjFKey: "FOREIGN KEY (task_id, story_id, project_id) REFERENCES tasks(id, story_id, project_id) ON DELETE RESTRICT/s/r",
	}
	gotForeignKeys := constraintShapes(t, pool, "effort_records", "f", `pg_get_constraintdef(oid) || '/' || confmatchtype::text || '/' || confdeltype::text`)
	if !reflect.DeepEqual(gotForeignKeys, wantForeignKeys) {
		t.Errorf("effort_records foreign keys = %v, want %v", gotForeignKeys, wantForeignKeys)
	}

	// Check constraints keep the table valid even for writers that bypass the application.
	wantChecks := map[string]string{
		effortHoursRangeCheck:     "CHECK (((hours > (0)::numeric) AND (hours <= (24)::numeric)))",
		effortDescriptionLenCheck: "CHECK (((char_length(description) >= 1) AND (char_length(description) <= 500)))",
	}
	gotChecks := constraintShapes(t, pool, "effort_records", "c", `pg_get_constraintdef(oid)`)
	if !reflect.DeepEqual(gotChecks, wantChecks) {
		t.Errorf("effort_records checks = %v, want %v", gotChecks, wantChecks)
	}

	// The unique keys that back the composite foreign keys live on the parent tables.
	wantMemberUnique := map[string]string{membersUniqueConstraint: "UNIQUE (id, project_id)"}
	if got := uniqueConstraintShapes(t, pool, "project_members", membersUniqueConstraint); !reflect.DeepEqual(got, wantMemberUnique) {
		t.Errorf("project_members unique key = %v, want %v", got, wantMemberUnique)
	}
	wantTaskUnique := map[string]string{tasksUniqueConstraint: "UNIQUE (id, story_id, project_id)"}
	if got := uniqueConstraintShapes(t, pool, "tasks", tasksUniqueConstraint); !reflect.DeepEqual(got, wantTaskUnique) {
		t.Errorf("tasks unique key = %v, want %v", got, wantTaskUnique)
	}

	wantIndexes := []string{
		"CREATE INDEX effort_records_member_idx ON effort_records USING btree (member_id)",
		"CREATE INDEX effort_records_project_story_idx ON effort_records USING btree (project_id, story_id, seq)",
		"CREATE INDEX effort_records_task_idx ON effort_records USING btree (task_id) WHERE (task_id IS NOT NULL)",
	}
	if got := effortIndexDefinitions(t, pool); !reflect.DeepEqual(got, wantIndexes) {
		t.Errorf("effort_records secondary indexes = %v, want %v", got, wantIndexes)
	}
}

// assertEffortSchemaAbsent verifies that a down migration removed everything it added.
func assertEffortSchemaAbsent(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if relationExists(t, pool, "effort_records") {
		t.Error("effort_records still exists after down")
	}
	if got := uniqueConstraintShapes(t, pool, "project_members", membersUniqueConstraint); len(got) != 0 {
		t.Errorf("project_members unique key still present after down: %v", got)
	}
	if got := uniqueConstraintShapes(t, pool, "tasks", tasksUniqueConstraint); len(got) != 0 {
		t.Errorf("tasks unique key still present after down: %v", got)
	}
}

// assertParentsIntact verifies the rows seeded by seedEffortParents survived the migration.
func assertParentsIntact(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	counts := map[string]int{"projects": 1, "stories": 1, "project_members": 1, "tasks": 1}
	for table, want := range counts {
		var got int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Errorf("%s rows = %d, want %d", table, got, want)
		}
	}
	var memberName, taskTitle string
	if err := pool.QueryRow(context.Background(), "SELECT full_name FROM project_members WHERE id = $1", effortMemberID).Scan(&memberName); err != nil || memberName != "Ana Perez" {
		t.Errorf("member after migration = (%q, %v), want Ana Perez", memberName, err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT title FROM tasks WHERE id = $1", effortTaskID).Scan(&taskTitle); err != nil || taskTitle != "Tarea" {
		t.Errorf("task after migration = (%q, %v), want Tarea", taskTitle, err)
	}
}

func constraintShapes(t *testing.T, pool *pgxpool.Pool, table, contype, shapeExpr string) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		fmt.Sprintf("SELECT conname, %s FROM pg_constraint WHERE conrelid = to_regclass($1) AND contype = $2", shapeExpr), table, contype)
	if err != nil {
		t.Fatalf("read %s constraints (%s): %v", table, contype, err)
	}
	defer rows.Close()
	shapes := make(map[string]string)
	for rows.Next() {
		var name, shape string
		if err := rows.Scan(&name, &shape); err != nil {
			t.Fatalf("scan %s constraint: %v", table, err)
		}
		shapes[name] = shape
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %s constraints: %v", table, err)
	}
	return shapes
}

// uniqueConstraintShapes returns the definition of the named UNIQUE constraint, or an empty map when absent.
func uniqueConstraintShapes(t *testing.T, pool *pgxpool.Pool, table, name string) map[string]string {
	t.Helper()
	shapes := make(map[string]string)
	if definition, ok := constraintShapes(t, pool, table, "u", `pg_get_constraintdef(oid)`)[name]; ok {
		shapes[name] = definition
	}
	return shapes
}

// effortIndexDefinitions returns the secondary index definitions (primary key excluded), sorted by name.
func effortIndexDefinitions(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename = 'effort_records' AND indexname <> 'effort_records_pkey'
		ORDER BY indexname
	`)
	if err != nil {
		t.Fatalf("read effort_records indexes: %v", err)
	}
	defer rows.Close()
	var definitions []string
	for rows.Next() {
		var definition string
		if err := rows.Scan(&definition); err != nil {
			t.Fatalf("scan effort_records index: %v", err)
		}
		definitions = append(definitions, normalizeIndexDefinition(definition))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate effort_records indexes: %v", err)
	}
	sort.Strings(definitions)
	return definitions
}

// schemaQualifier matches the schema prefix that pg_indexes adds to the table name.
var schemaQualifier = regexp.MustCompile(`ON [a-z0-9_]+\.effort_records`)

// normalizeIndexDefinition drops the schema qualifier so definitions compare equal in isolated schemas.
func normalizeIndexDefinition(definition string) string {
	return schemaQualifier.ReplaceAllString(definition, "ON effort_records")
}

func relationExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), "SELECT to_regclass($1) IS NOT NULL", name).Scan(&exists); err != nil {
		t.Fatalf("inspect relation %s: %v", name, err)
	}
	return exists
}

func effortRecordCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM effort_records").Scan(&count); err != nil {
		t.Fatalf("count effort_records: %v", err)
	}
	return count
}

// seedEffortParents inserts one project, one story assigned to a sprint, one member and one task.
func seedEffortParents(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	insertProject(t, pool, effortProjectAID)
	insertSprint(t, pool, effortSprintID, effortProjectAID)
	insertStory(t, pool, effortStoryID, effortProjectAID)
	assignStory(t, pool, effortSprintID, effortStoryID, effortProjectAID)
	insertMember(t, pool, effortMemberID, effortProjectAID, "Ana Perez")
	insertTask(t, pool, effortTaskID, effortProjectAID, effortSprintID, effortStoryID)
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

func insertSprint(t *testing.T, pool *pgxpool.Pool, id, projectID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Delivery')", id, projectID); err != nil {
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
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprint_stories (sprint_id, story_id, project_id) VALUES ($1, $2, $3)", sprintID, storyID, projectID); err != nil {
		t.Fatalf("assign story to sprint: %v", err)
	}
}

func insertMember(t *testing.T, pool *pgxpool.Pool, id, projectID, fullName string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "INSERT INTO project_members (id, project_id, full_name) VALUES ($1, $2, $3)", id, projectID, fullName); err != nil {
		t.Fatalf("insert project member: %v", err)
	}
}

func insertTask(t *testing.T, pool *pgxpool.Pool, id, projectID, sprintID, storyID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "INSERT INTO tasks (id, project_id, sprint_id, story_id, title) VALUES ($1, $2, $3, $4, 'Tarea')", id, projectID, sprintID, storyID); err != nil {
		t.Fatalf("insert task: %v", err)
	}
}

// insertEffortRecord inserts a valid record over the seeded story, optionally over a task.
func insertEffortRecord(t *testing.T, pool *pgxpool.Pool, id string, taskID *string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO effort_records (id, project_id, story_id, task_id, member_id, work_date, description, hours)
		VALUES ($1, $2, $3, $4, $5, '2026-03-10', 'Revision de codigo', 2.5)
	`, id, effortProjectAID, effortStoryID, taskID, effortMemberID)
	if err != nil {
		t.Fatalf("insert effort record: %v", err)
	}
}

// effortDatabase returns a database with every canonical migration (000001-000012) applied.
func effortDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return effortDatabaseAtVersion(t, 12)
}

// effortDatabaseAtVersion returns a database migrated up to and including the given version.
func effortDatabaseAtVersion(t *testing.T, version int) *pgxpool.Pool {
	t.Helper()
	pool := openEffortPool(t)
	for v := 1; v <= version; v++ {
		applyEffortMigration(t, pool, fmt.Sprintf("%06d", v), "up")
	}
	return pool
}

func openEffortPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	if dsn := os.Getenv("EFFORT_TEST_DATABASE_URL"); dsn != "" {
		return testpostgres.OpenIsolated(t, dsn, "EFFORT_TEST_DATABASE_URL", "effort_test")
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("effort_test"),
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
	waitForEffortDatabaseReady(t, ctx, pool)
	return pool
}

// waitForEffortDatabaseReady polls with Ping until the container accepts connections.
// The container's "ready" log line fires before PostgreSQL finishes starting up;
// connecting immediately can fail with an EOF on the wire.
func waitForEffortDatabaseReady(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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

// applyEffortMigration executes the single migration file matching version and direction
// ("up" or "down") from the shared migrations directory.
func applyEffortMigration(t *testing.T, pool *pgxpool.Pool, version, direction string) {
	t.Helper()
	pattern := filepath.Join(effortModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", version+"_*."+direction+".sql")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("match migration %s %s: %v", version, direction, err)
	}
	if len(matches) != 1 {
		t.Fatalf("migration %s %s: found %d files (%v), want exactly 1", version, direction, len(matches), matches)
	}
	migration, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read migration %s: %v", matches[0], err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply migration %s: %v", filepath.Base(matches[0]), err)
	}
}

func effortModuleRoot(t *testing.T) string {
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
