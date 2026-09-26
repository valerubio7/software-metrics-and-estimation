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

const otherProjectID = "0b5cbb21-6a3e-4d4b-9b56-7d2f5d5f1c11"

// storedRow reads a story straight from SQL, independently of the repository under test.
// Hours keep their stored NUMERIC text so scale is observable ("8.00", not 8).
type storedRow struct {
	ID, ProjectID, Title, Description, Priority, Status string
	Points                                              *int
	Criteria                                            []string
	Hours                                               *string
}

func readRow(t *testing.T, pool *pgxpool.Pool, id string) storedRow {
	t.Helper()
	var row storedRow
	err := pool.QueryRow(context.Background(), `
		SELECT id::text, project_id::text, title, description, priority, status,
		       story_points, acceptance_criteria, estimated_hours::text
		FROM stories WHERE id = $1
	`, id).Scan(&row.ID, &row.ProjectID, &row.Title, &row.Description, &row.Priority, &row.Status,
		&row.Points, &row.Criteria, &row.Hours)
	if err != nil {
		t.Fatalf("read stored row %s: %v", id, err)
	}
	return row
}

// seedStory inserts a story by direct SQL so story_points and estimated_hours can be
// seeded (the repository never writes story_points). Empty hours mean SQL NULL.
func seedStory(t *testing.T, pool *pgxpool.Pool, story domain.Story, points *int, hours string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO stories (id, project_id, title, description, priority, status, story_points, acceptance_criteria, estimated_hours)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::numeric)
	`, story.ID, story.ProjectID, story.Title, story.Description, story.Priority, story.Status,
		points, story.AcceptanceCriteria, hours)
	if err != nil {
		t.Fatalf("seed story: %v", err)
	}
}

// show renders a nullable value readably in failure messages instead of a pointer address.
func show[T any](p *T) any {
	if p == nil {
		return "nil"
	}
	return *p
}

func hoursPtr(v float64) *float64 { return &v }

func intPtr(v int) *int { return &v }

func textPtr(v string) *string { return &v }

// editedStory is a valid full replacement of the story seeded by validStory.
func editedStory(hours *float64) domain.Story {
	return domain.Story{
		ID: storyID, ProjectID: projectID, Title: "  Titulo nuevo  ", Description: "Descripcion nueva",
		Priority: "alta", Status: "en_progreso", AcceptanceCriteria: []string{"Tercero", " Primero ", "Segundo"},
		EstimatedHours: hours,
	}
}

func TestStoryRepositoryUpdateRoundTripsEstimatedHours(t *testing.T) {
	for _, tc := range []struct {
		name     string
		hours    float64
		wantText string
	}{
		{name: "lower bound", hours: 0.01, wantText: "0.01"},
		{name: "upper bound", hours: 99999.99, wantText: "99999.99"},
		{name: "integer", hours: 8, wantText: "8.00"},
		{name: "one decimal", hours: 2.5, wantText: "2.50"},
		{name: "float-hostile decimal", hours: 0.07, wantText: "0.07"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := storyDatabase(t)
			insertProject(t, pool, projectID)
			seedStory(t, pool, validStory(projectID), nil, "")

			got, err := storypostgres.NewPostgresStoryRepository(pool).Update(context.Background(), editedStory(hoursPtr(tc.hours)))
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}
			if got.EstimatedHours == nil || *got.EstimatedHours != tc.hours {
				t.Errorf("RETURNING estimated_hours = %v, want %v", show(got.EstimatedHours), tc.hours)
			}
			if got.StoryPoints != nil {
				t.Errorf("RETURNING story_points = %v, want nil (NULL scanned into *int)", *got.StoryPoints)
			}
			if row := readRow(t, pool, storyID); row.Hours == nil || *row.Hours != tc.wantText {
				t.Errorf("stored estimated_hours = %v, want %s", show(row.Hours), tc.wantText)
			}
		})
	}
}

func TestStoryRepositoryUpdateReplacesAllEditableFields(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	seed := validStory(projectID)
	seed.AcceptanceCriteria = []string{"Viejo uno", "Viejo dos", "Viejo tres", "Viejo cuatro"}
	seedStory(t, pool, seed, nil, "")
	edited := editedStory(hoursPtr(12.5))

	got, err := storypostgres.NewPostgresStoryRepository(pool).Update(context.Background(), edited)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !reflect.DeepEqual(got, edited) {
		t.Errorf("Update() = %#v, want %#v", got, edited)
	}
	want := storedRow{
		ID: storyID, ProjectID: projectID, Title: edited.Title, Description: edited.Description,
		Priority: "alta", Status: "en_progreso", Criteria: []string{"Tercero", " Primero ", "Segundo"},
		Hours: textPtr("12.50"),
	}
	if row := readRow(t, pool, storyID); !reflect.DeepEqual(row, want) {
		t.Errorf("stored row = %#v, want %#v (criteria order kept, previous criteria gone)", row, want)
	}
}

func TestStoryRepositoryUpdateWithNilHoursClearsEstimateToNull(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	seedStory(t, pool, validStory(projectID), nil, "8")

	got, err := storypostgres.NewPostgresStoryRepository(pool).Update(context.Background(), editedStory(nil))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if got.EstimatedHours != nil {
		t.Errorf("RETURNING estimated_hours = %v, want nil", *got.EstimatedHours)
	}
	var isNull bool
	if err := pool.QueryRow(context.Background(), "SELECT estimated_hours IS NULL FROM stories WHERE id = $1", storyID).Scan(&isNull); err != nil {
		t.Fatalf("read estimated_hours: %v", err)
	}
	if !isNull {
		t.Error("estimated_hours is not SQL NULL after clearing (a stored 0 would violate the CHECK)")
	}
}

func TestStoryRepositoryUpdateKeepsIdentityAndStoryPoints(t *testing.T) {
	for _, tc := range []struct {
		name   string
		points *int
	}{
		{name: "estimated story", points: intPtr(5)},
		{name: "larger estimate with hours", points: intPtr(13)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := storyDatabase(t)
			insertProject(t, pool, projectID)
			seedStory(t, pool, validStory(projectID), tc.points, "")

			got, err := storypostgres.NewPostgresStoryRepository(pool).Update(context.Background(), editedStory(hoursPtr(1.5)))
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			if got.ID != storyID || got.ProjectID != projectID {
				t.Errorf("Update() identity = (%s, %s), want (%s, %s)", got.ID, got.ProjectID, storyID, projectID)
			}
			if got.StoryPoints == nil || *got.StoryPoints != *tc.points {
				t.Errorf("Update() story_points = %v, want %d from storage", show(got.StoryPoints), *tc.points)
			}
			row := readRow(t, pool, storyID)
			if row.ProjectID != projectID || row.Points == nil || *row.Points != *tc.points {
				t.Errorf("stored project/points = (%s, %v), want (%s, %d)", row.ProjectID, show(row.Points), projectID, *tc.points)
			}
		})
	}
}

func TestStoryRepositoryUpdateReportsNotFoundWithoutTouchingData(t *testing.T) {
	t.Run("story of another project", func(t *testing.T) {
		pool := storyDatabase(t)
		insertProject(t, pool, projectID)
		insertProject(t, pool, otherProjectID)
		seedStory(t, pool, validStory(projectID), intPtr(5), "8")
		before := readRow(t, pool, storyID)
		update := editedStory(hoursPtr(1))
		update.ProjectID = otherProjectID

		got, err := storypostgres.NewPostgresStoryRepository(pool).Update(context.Background(), update)

		if !errors.Is(err, application.ErrStoryNotFound) {
			t.Errorf("Update(wrong project) error = %v, want ErrStoryNotFound", err)
		}
		if !reflect.DeepEqual(got, domain.Story{}) {
			t.Errorf("Update(wrong project) story = %#v, want zero value", got)
		}
		if after := readRow(t, pool, storyID); !reflect.DeepEqual(after, before) {
			t.Errorf("row after cross-project update = %#v, want unchanged %#v", after, before)
		}
	})

	t.Run("missing story", func(t *testing.T) {
		pool := storyDatabase(t)
		insertProject(t, pool, projectID)

		_, err := storypostgres.NewPostgresStoryRepository(pool).Update(context.Background(), editedStory(nil))

		if !errors.Is(err, application.ErrStoryNotFound) {
			t.Errorf("Update(missing story) error = %v, want ErrStoryNotFound", err)
		}
		if got := storyCount(t, pool); got != 0 {
			t.Errorf("story count = %d, want 0 (update must not create rows)", got)
		}
	})

	t.Run("unrelated database errors are not labeled as not found", func(t *testing.T) {
		pool := storyDatabase(t)
		insertProject(t, pool, projectID)
		seedStory(t, pool, validStory(projectID), nil, "")
		repository := storypostgres.NewPostgresStoryRepository(pool)

		canceled, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := repository.Update(canceled, editedStory(nil))
		if err == nil || errors.Is(err, application.ErrStoryNotFound) || !errors.Is(err, context.Canceled) {
			t.Errorf("Update(canceled context) error = %v, want context.Canceled and not ErrStoryNotFound", err)
		}

		invalid := editedStory(nil)
		invalid.Status = "cancelada"
		_, err = repository.Update(context.Background(), invalid)
		assertDatabaseError(t, err, "23514", "stories_status_check")
		if errors.Is(err, application.ErrStoryNotFound) {
			t.Errorf("CHECK violation mislabeled as not found: %v", err)
		}
		if row := readRow(t, pool, storyID); row.Status != "pendiente" {
			t.Errorf("status after rejected update = %q, want pendiente", row.Status)
		}
	})
}

func TestStoryRepositoryEnforcesNamedStoryConstraints(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	seedStory(t, pool, validStory(projectID), nil, "")

	for _, tc := range []struct {
		name       string
		constraint string
		wantParts  []string
	}{
		{name: "status set", constraint: "stories_status_check", wantParts: []string{"status", "'pendiente'", "'en_progreso'", "'completada'"}},
		{name: "positive estimate", constraint: "stories_estimated_hours_positive", wantParts: []string{"estimated_hours > "}},
	} {
		var definition string
		err := pool.QueryRow(context.Background(), `
			SELECT pg_get_constraintdef(oid) FROM pg_constraint
			WHERE conrelid = 'stories'::regclass AND conname = $1
		`, tc.constraint).Scan(&definition)
		if err != nil {
			t.Fatalf("%s: named constraint %s missing: %v", tc.name, tc.constraint, err)
		}
		for _, part := range tc.wantParts {
			if !strings.Contains(definition, part) {
				t.Errorf("%s: %s = %q, want it to contain %q", tc.name, tc.constraint, definition, part)
			}
		}
	}

	for _, tc := range []struct {
		name       string
		statement  string
		code       string
		constraint string
	}{
		{name: "status outside the set", statement: "UPDATE stories SET status = 'cancelada'", code: "23514", constraint: "stories_status_check"},
		{name: "zero estimate", statement: "UPDATE stories SET estimated_hours = 0", code: "23514", constraint: "stories_estimated_hours_positive"},
		{name: "negative estimate", statement: "UPDATE stories SET estimated_hours = -1", code: "23514", constraint: "stories_estimated_hours_positive"},
		{name: "estimate above NUMERIC(7,2)", statement: "UPDATE stories SET estimated_hours = 100000", code: "22003", constraint: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pool.Exec(context.Background(), tc.statement)
			assertDatabaseError(t, err, tc.code, tc.constraint)
		})
	}
}

func TestStoryRepositoryUpdateTreatsHostileTextAsData(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	const bystanderID = "331a7ff5-12eb-4b2d-83fd-b51fd77e413d"
	bystander := validStory(projectID)
	bystander.ID = bystanderID
	seedStory(t, pool, validStory(projectID), nil, "")
	seedStory(t, pool, bystander, nil, "3")
	bystanderBefore := readRow(t, pool, bystanderID)
	hostile := editedStory(hoursPtr(2))
	hostile.Title = `x'); DROP TABLE stories; --`
	hostile.Description = `'; UPDATE stories SET title = 'pwned' WHERE '1'='1`
	hostile.AcceptanceCriteria = []string{`"quoted"; --`, `it's $1 \ done`}

	got, err := storypostgres.NewPostgresStoryRepository(pool).Update(context.Background(), hostile)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !reflect.DeepEqual(got, hostile) {
		t.Errorf("Update() = %#v, want %#v (hostile text stored verbatim)", got, hostile)
	}
	if after := readRow(t, pool, bystanderID); !reflect.DeepEqual(after, bystanderBefore) {
		t.Errorf("bystander row = %#v, want unchanged %#v", after, bystanderBefore)
	}
	if got := storyCount(t, pool); got != 2 {
		t.Errorf("story count = %d, want 2 (table intact)", got)
	}
}

func TestStoryRepositoryUpdateTriangulation(t *testing.T) {
	t.Run("same update twice is idempotent", func(t *testing.T) {
		pool := storyDatabase(t)
		insertProject(t, pool, projectID)
		seedStory(t, pool, validStory(projectID), intPtr(3), "")
		repository := storypostgres.NewPostgresStoryRepository(pool)
		edited := editedStory(hoursPtr(4.25))

		first, err := repository.Update(context.Background(), edited)
		if err != nil {
			t.Fatalf("first Update() error = %v", err)
		}
		second, err := repository.Update(context.Background(), edited)
		if err != nil {
			t.Fatalf("second Update() error = %v", err)
		}

		if !reflect.DeepEqual(first, second) {
			t.Errorf("second Update() = %#v, want identical %#v", second, first)
		}
	})

	t.Run("last of two consecutive updates wins", func(t *testing.T) {
		pool := storyDatabase(t)
		insertProject(t, pool, projectID)
		seedStory(t, pool, validStory(projectID), nil, "")
		repository := storypostgres.NewPostgresStoryRepository(pool)
		if _, err := repository.Update(context.Background(), editedStory(hoursPtr(9))); err != nil {
			t.Fatalf("first Update() error = %v", err)
		}
		last := editedStory(nil)
		last.Title = "Ultimo titulo"
		last.AcceptanceCriteria = []string{"Solo uno"}

		got, err := repository.Update(context.Background(), last)
		if err != nil {
			t.Fatalf("second Update() error = %v", err)
		}

		if !reflect.DeepEqual(got, last) {
			t.Errorf("Update() = %#v, want %#v", got, last)
		}
		if row := readRow(t, pool, storyID); row.Title != "Ultimo titulo" || row.Hours != nil || !reflect.DeepEqual(row.Criteria, []string{"Solo uno"}) {
			t.Errorf("stored row = %#v, want the last update only", row)
		}
	})

	t.Run("every status of the closed set persists", func(t *testing.T) {
		for _, status := range domain.AllowedStatuses() {
			t.Run(status, func(t *testing.T) {
				pool := storyDatabase(t)
				insertProject(t, pool, projectID)
				seedStory(t, pool, validStory(projectID), nil, "")
				edited := editedStory(nil)
				edited.Status = status

				got, err := storypostgres.NewPostgresStoryRepository(pool).Update(context.Background(), edited)
				if err != nil {
					t.Fatalf("Update(%s) error = %v", status, err)
				}
				if got.Status != status || readRow(t, pool, storyID).Status != status {
					t.Errorf("status = (%q returned, %q stored), want %q", got.Status, readRow(t, pool, storyID).Status, status)
				}
			})
		}
	})
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
	for _, name := range []string{
		"000001_create_projects.up.sql",
		"000002_create_stories.up.sql",
		"000003_add_story_estimated_hours.up.sql",
	} {
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
