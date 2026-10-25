package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/tests/integration/testpostgres"
)

const (
	defectProjectAID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	defectSprintID   = "b2a6a455-06e2-41ee-b011-7462c71375a0"
	defectStoryID    = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	defectRecordID   = "f47ac10b-58cc-4372-a567-0e02b2c3d479"

	// Segundo proyecto con Sprint e historia propios para probar el alcance por proyecto.
	defectProjectBID = "3c6c2a7d-5ea1-4f40-8b8d-9d4e5f607182"
	defectSprintBID  = "4d7d3b8e-6fb2-4051-9c9e-ae5f60718293"
	defectStoryBID   = "5e8e4c9f-70c3-4162-8daf-bf60718293a4"
)

// Historia del proyecto A que no ha sido asignada a ningún Sprint (solo backlog).
const defectBacklogStoryID = "2b5b1f6c-4d90-4e3f-9a7c-8c3d4e5f6071"

const unknownStoryID = "70a06eb1-92e5-4384-8fc1-d18293a4b5c6"

const (
	defectMigrationVersion = "000013"
	lastPreDefectMigration = 12

	defectStoryProjectFKey  = "defects_story_project_fkey"
	defectSprintProjectFKey = "defects_sprint_project_fkey"

	defectSeverityCheck     = "defects_severity_check"
	defectStatusCheck       = "defects_status_check"
	defectDescriptionLength = "defects_description_length"
)

func TestDefectMigrationCreatesSchema(t *testing.T) {
	pool := defectDatabase(t)

	assertDefectSchema(t, pool)
}

func TestDefectMigrationUpOverVersion12KeepsExistingData(t *testing.T) {
	pool := defectDatabaseAtVersion(t, lastPreDefectMigration)
	seedDefectParents(t, pool)
	if exists := relationExists(t, pool, "defects"); exists {
		t.Fatal("defects must not exist at schema version 12")
	}

	applyDefectMigration(t, pool, defectMigrationVersion, "up")

	assertDefectSchema(t, pool)
	assertDefectParentsIntact(t, pool)
	if got := defectCount(t, pool); got != 0 {
		t.Errorf("defects rows after up = %d, want 0", got)
	}
}

func TestDefectMigrationDownRemovesSchemaAndKeepsData(t *testing.T) {
	pool := defectDatabase(t)
	seedDefectParents(t, pool)
	insertDefect(t, pool, defectRecordID, defectProjectAID, defectStoryID, defectSprintID)

	applyDefectMigration(t, pool, defectMigrationVersion, "down")

	assertDefectSchemaAbsent(t, pool)
	assertDefectParentsIntact(t, pool)
}

func TestDefectMigrationDownIsIdempotent(t *testing.T) {
	pool := defectDatabase(t)
	seedDefectParents(t, pool)

	applyDefectMigration(t, pool, defectMigrationVersion, "down")
	applyDefectMigration(t, pool, defectMigrationVersion, "down")

	assertDefectSchemaAbsent(t, pool)
	assertDefectParentsIntact(t, pool)
}

func TestDefectMigrationCanBeReapplied(t *testing.T) {
	pool := defectDatabase(t)
	seedDefectParents(t, pool)
	applyDefectMigration(t, pool, defectMigrationVersion, "down")

	applyDefectMigration(t, pool, defectMigrationVersion, "up")

	assertDefectSchema(t, pool)
	assertDefectParentsIntact(t, pool)
	insertDefect(t, pool, defectRecordID, defectProjectAID, defectStoryID, defectSprintID)
	if got := defectCount(t, pool); got != 1 {
		t.Errorf("defects rows after reapplied up = %d, want 1", got)
	}
}

// defectCheckCase describes one row that may violate a CHECK constraint (empty wantCheck
// means the row must be accepted because the full catalog exists in the database).
type defectCheckCase struct {
	severity    string
	status      string
	description string
	wantCheck   string
}

type defectFKCase struct {
	storyID        string
	sprintID       string
	wantConstraint string
}

// sortedKeys returns the map keys in a deterministic order so subtests are stable.
func sortedKeys[V any](cases map[string]V) []string {
	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestDefectMigrationCheckConstraintsRejectInvalidRows(t *testing.T) {
	pool := defectDatabase(t)
	seedDefectParents(t, pool)

	cases := map[string]defectCheckCase{
		"severity_out_of_catalog":          {severity: "grave", status: "detectado", description: "El login acepta contraseñas vacías", wantCheck: defectSeverityCheck},
		"severity_empty":                   {severity: "", status: "detectado", description: "El login acepta contraseñas vacías", wantCheck: defectSeverityCheck},
		"status_out_of_catalog":            {severity: "alta", status: "nuevo", description: "El login acepta contraseñas vacías", wantCheck: defectStatusCheck},
		"status_empty":                     {severity: "alta", status: "", description: "El login acepta contraseñas vacías", wantCheck: defectStatusCheck},
		"status_resuelto_is_valid_storage": {severity: "alta", status: "resuelto", description: "El login acepta contraseñas vacías", wantCheck: ""},
		"description_too_long":             {severity: "alta", status: "detectado", description: strings.Repeat("x", 501), wantCheck: defectDescriptionLength},
	}

	for index, name := range sortedKeys(cases) {
		tc := cases[name]
		t.Run(name, func(t *testing.T) {
			id := fmt.Sprintf("00000000-0000-0000-0000-%012d", index+1)
			_, err := pool.Exec(context.Background(), `
				INSERT INTO defects (id, project_id, story_id, sprint_id, description, severity, status)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, id, defectProjectAID, defectStoryID, defectSprintID, tc.description, tc.severity, tc.status)

			if tc.wantCheck == "" {
				// El catálogo completo vive en la base: 'resuelto' es un valor válido de almacenamiento.
				if err != nil {
					t.Fatalf("insert defect with full-catalog status: %v", err)
				}
				if got := defectCount(t, pool); got != 1 {
					t.Errorf("defects rows after insert with status resuelto = %d, want 1", got)
				}
				if _, err := pool.Exec(context.Background(), "DELETE FROM defects WHERE id = $1", id); err != nil {
					t.Fatalf("delete full-catalog row: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("insert invalid defect: expected failure")
			}
			constraint, ok := checkViolationConstraint(err)
			if !ok {
				t.Fatalf("insert invalid defect error is not a check violation: %v", err)
			}
			if constraint != tc.wantCheck {
				t.Errorf("check violation constraint = %q, want %q", constraint, tc.wantCheck)
			}
			if got := defectCount(t, pool); got != 0 {
				t.Errorf("defects rows after failed insert = %d, want 0", got)
			}
		})
	}
}

func TestDefectMigrationForeignKeyConstraintsScopeReferences(t *testing.T) {
	pool := defectDatabase(t)
	seedDefectParents(t, pool)

	cases := map[string]defectFKCase{
		"story_of_other_project":    {storyID: defectStoryBID, sprintID: defectSprintID, wantConstraint: defectStoryProjectFKey},
		"nonexistent_story":         {storyID: unknownStoryID, sprintID: defectSprintID, wantConstraint: defectStoryProjectFKey},
		"sprint_of_other_project":   {storyID: defectStoryID, sprintID: defectSprintBID, wantConstraint: defectSprintProjectFKey},
		"story_and_sprint_of_other": {storyID: defectStoryBID, sprintID: defectSprintBID, wantConstraint: defectStoryProjectFKey},
	}

	for index, name := range sortedKeys(cases) {
		tc := cases[name]
		t.Run(name, func(t *testing.T) {
			_, err := pool.Exec(context.Background(), `
				INSERT INTO defects (id, project_id, story_id, sprint_id, description, severity, status)
				VALUES ($1, $2, $3, $4, 'El login acepta contraseñas vacías', 'alta', 'detectado')
			`, fmt.Sprintf("00000000-0000-0000-0000-%012d", index+1), defectProjectAID, tc.storyID, tc.sprintID)
			if err == nil {
				t.Fatal("insert defect with crossed references: expected failure")
			}
			constraint, ok := foreignKeyViolationConstraint(err)
			if !ok {
				t.Fatalf("insert defect with crossed references error is not a foreign key violation: %v", err)
			}
			if constraint != tc.wantConstraint {
				t.Errorf("foreign key violation constraint = %q, want %q", constraint, tc.wantConstraint)
			}
			if got := defectCount(t, pool); got != 0 {
				t.Errorf("defects rows after failed insert = %d, want 0", got)
			}
		})
	}
}

// assertDefectSchema verifies the full defects contract by reading the catalog.
func assertDefectSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	wantColumns := map[string]string{
		"id":          "uuid/NO",
		"project_id":  "uuid/NO",
		"story_id":    "uuid/NO",
		"sprint_id":   "uuid/NO",
		"description": "text/NO",
		"severity":    "text/NO",
		"status":      "text/NO",
		"seq":         "bigint/NO",
		"created_at":  "timestamp with time zone/NO",
	}
	gotColumns := make(map[string]string)
	rows, err := pool.Query(context.Background(), `
		SELECT a.attname,
		       format_type(a.atttypid, a.atttypmod) || '/' || CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END
		FROM pg_attribute a
		WHERE a.attrelid = to_regclass('defects') AND a.attnum > 0 AND NOT a.attisdropped
	`)
	if err != nil {
		t.Fatalf("read defects columns: %v", err)
	}
	for rows.Next() {
		var name, shape string
		if err := rows.Scan(&name, &shape); err != nil {
			rows.Close()
			t.Fatalf("scan defects column: %v", err)
		}
		gotColumns[name] = shape
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate defects columns: %v", err)
	}
	if !reflect.DeepEqual(gotColumns, wantColumns) {
		t.Errorf("defects columns = %v, want %v", gotColumns, wantColumns)
	}

	// Foreign keys: composite, MATCH SIMPLE (confmatchtype 's') and ON DELETE RESTRICT (confdeltype 'r').
	wantForeignKeys := map[string]string{
		defectStoryProjectFKey:  "FOREIGN KEY (story_id, project_id) REFERENCES stories(id, project_id) ON DELETE RESTRICT/s/r",
		defectSprintProjectFKey: "FOREIGN KEY (sprint_id, project_id) REFERENCES sprints(id, project_id) ON DELETE RESTRICT/s/r",
	}
	gotForeignKeys := constraintShapes(t, pool, "defects", "f", `pg_get_constraintdef(oid) || '/' || confmatchtype::text || '/' || confdeltype::text`)
	if !reflect.DeepEqual(gotForeignKeys, wantForeignKeys) {
		t.Errorf("defects foreign keys = %v, want %v", gotForeignKeys, wantForeignKeys)
	}

	// Check constraints keep the table valid even for writers that bypass the application.
	// Catálogos y longitud; las formas se comparan canónicas vía canonicalConstraintDef.
	wantChecks := map[string]string{
		defectSeverityCheck:     "CHECK((severity=ANY(ARRAY['alta','media','baja'])))",
		defectStatusCheck:       "CHECK((status=ANY(ARRAY['detectado','en_correccion','resuelto'])))",
		defectDescriptionLength: "CHECK(((char_length(description)>=1)AND(char_length(description)<=500)))",
	}
	gotChecks := make(map[string]string)
	for name, def := range constraintShapes(t, pool, "defects", "c", `pg_get_constraintdef(oid)`) {
		gotChecks[name] = canonicalConstraintDef(t, def)
	}
	if !reflect.DeepEqual(gotChecks, wantChecks) {
		t.Errorf("defects checks = %v, want %v", gotChecks, wantChecks)
	}

	wantIndexes := []string{
		"CREATE INDEX defects_project_seq_idx ON defects USING btree (project_id, seq)",
		"CREATE INDEX defects_project_sprint_idx ON defects USING btree (project_id, sprint_id, seq)",
		"CREATE INDEX defects_project_story_idx ON defects USING btree (project_id, story_id, seq)",
	}
	if got := defectIndexDefinitions(t, pool); !reflect.DeepEqual(got, wantIndexes) {
		t.Errorf("defects secondary indexes = %v, want %v", got, wantIndexes)
	}
}

// assertDefectSchemaAbsent verifies that a down migration removed everything it added.
func assertDefectSchemaAbsent(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if relationExists(t, pool, "defects") {
		t.Error("defects still exists after down")
	}
}

// assertDefectParentsIntact verifies the rows seeded by seedDefectParents survived the migration.
func assertDefectParentsIntact(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	assertDefectParentsIntactN(t, pool, 2, 3, 2)
	var sprintGoal string
	if err := pool.QueryRow(context.Background(), "SELECT sprint_goal FROM sprints WHERE id = $1", defectSprintID).Scan(&sprintGoal); err != nil || sprintGoal != "Delivery" {
		t.Errorf("sprint after migration = (%q, %v), want Delivery", sprintGoal, err)
	}
}

// defectIndexDefinitions returns the secondary index definitions (primary key excluded).
func defectIndexDefinitions(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename = 'defects' AND indexname <> 'defects_pkey'
		ORDER BY indexname
	`)
	if err != nil {
		t.Fatalf("read defects indexes: %v", err)
	}
	defer rows.Close()
	var definitions []string
	for rows.Next() {
		var definition string
		if err := rows.Scan(&definition); err != nil {
			t.Fatalf("scan defects index: %v", err)
		}
		definitions = append(definitions, normalizeIndexDefinition(definition))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate defects indexes: %v", err)
	}
	sort.Strings(definitions)
	return definitions
}

// schemaQualifier matches the schema prefix that pg_indexes adds to the table name.
var schemaQualifier = regexp.MustCompile(`ON [a-z0-9_]+\.defects`)

// normalizeIndexDefinition drops the schema qualifier so definitions compare equal in isolated schemas.
func normalizeIndexDefinition(definition string) string {
	return schemaQualifier.ReplaceAllString(definition, "ON defects")
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

// canonicalConstraintDef normalizes a constraint definition so version-specific naming
// (text casts, spacing, letter casing) does not hide the comparison.
func canonicalConstraintDef(t *testing.T, def string) string {
	t.Helper()
	normalized := strings.ReplaceAll(def, "::text", "")
	normalized = strings.ReplaceAll(normalized, "::", "")
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, "")
	return normalized
}

// relationExists reports whether the named relation exists in the current schema.
func relationExists(t *testing.T, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), "SELECT to_regclass($1) IS NOT NULL", name).Scan(&exists); err != nil {
		t.Fatalf("inspect relation %s: %v", name, err)
	}
	return exists
}

func defectCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM defects").Scan(&count); err != nil {
		t.Fatalf("count defects: %v", err)
	}
	return count
}

// seedDefectParents inserts the two projects with their Sprints and stories so the tests
// can prove that references stay scoped by project.
func seedDefectParents(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	insertProject(t, pool, defectProjectAID)
	insertSprint(t, pool, defectSprintID, defectProjectAID)
	insertStory(t, pool, defectStoryID, defectProjectAID)
	insertStory(t, pool, defectBacklogStoryID, defectProjectAID)
	insertProject(t, pool, defectProjectBID)
	insertSprint(t, pool, defectSprintBID, defectProjectBID)
	insertStory(t, pool, defectStoryBID, defectProjectBID)
	assertDefectParentsIntactN(t, pool, 2, 3, 2)
}

// assertDefectParentsIntactN verifies exact counts of parents (projects, stories, sprints).
func assertDefectParentsIntactN(t *testing.T, pool *pgxpool.Pool, projects, stories, sprints int) {
	t.Helper()
	for table, want := range map[string]int{"projects": projects, "stories": stories, "sprints": sprints} {
		var got int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Errorf("%s rows = %d, want %d", table, got, want)
		}
	}
}

// insertProject inserts the parent project row used by defect tests.
func insertProject(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO projects (id, name, start_date, planned_finish_date)
		VALUES ($1, 'Digital', '2026-03-01', '2026-06-30')
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

// insertDefect inserts a valid defect row inside the seeded project.
func insertDefect(t *testing.T, pool *pgxpool.Pool, id, projectID, storyID, sprintID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO defects (id, project_id, story_id, sprint_id, description, severity, status)
		VALUES ($1, $2, $3, $4, 'El login acepta contraseñas vacías', 'alta', 'detectado')
	`, id, projectID, storyID, sprintID)
	if err != nil {
		t.Fatalf("insert defect: %v", err)
	}
}

// defectDatabase returns a database with every canonical migration (000001-000013) applied.
func defectDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return defectDatabaseAtVersion(t, 13)
}

// defectDatabaseAtVersion returns a database migrated up to and including the given version.
func defectDatabaseAtVersion(t *testing.T, version int) *pgxpool.Pool {
	t.Helper()
	pool := openDefectPool(t)
	for v := 1; v <= version; v++ {
		applyDefectMigration(t, pool, fmt.Sprintf("%06d", v), "up")
	}
	return pool
}

// foreignKeyViolationConstraint extracts the constraint name of a 23503 error.
func foreignKeyViolationConstraint(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		return "", false
	}
	return pgErr.ConstraintName, true
}

// checkViolationConstraint extracts the constraint name of a 23514 error.
func checkViolationConstraint(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		return "", false
	}
	return pgErr.ConstraintName, true
}

func applyDefectMigration(t *testing.T, pool *pgxpool.Pool, version, direction string) {
	t.Helper()
	pattern := filepath.Join(defectModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", version+"_*."+direction+".sql")
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

func openDefectPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	if dsn := os.Getenv("DEFECT_TEST_DATABASE_URL"); dsn != "" {
		return testpostgres.OpenIsolated(t, dsn, "DEFECT_TEST_DATABASE_URL", "defect_test")
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("defect_test"),
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
	waitForDefectDatabaseReady(t, ctx, pool)
	return pool
}

// waitForDefectDatabaseReady polls with Ping until the container accepts connections.
// The container's "ready" log line fires before PostgreSQL finishes starting up;
// connecting immediately can fail with an EOF on the wire.
func waitForDefectDatabaseReady(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
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

func defectModuleRoot(t *testing.T) string {
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
