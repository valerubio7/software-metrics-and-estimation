package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
)

func TestUS09SchemaRequiresCompleteMigration(t *testing.T) {
	pool := storyDatabase(t)
	var association, closed bool
	err := pool.QueryRow(context.Background(), `
		SELECT to_regclass('sprint_stories') IS NOT NULL,
		       EXISTS (SELECT 1 FROM information_schema.columns
		               WHERE table_schema = current_schema()
		                 AND table_name = 'sprints' AND column_name = 'is_closed')
	`).Scan(&association, &closed)
	if err != nil {
		t.Fatalf("inspect assignment schema: %v", err)
	}
	if !association || !closed {
		t.Fatalf("incomplete US09 schema: sprint_stories=%v sprints.is_closed=%v", association, closed)
	}
}

const sprintID = "b2a6a455-06e2-41ee-b011-7462c71375a0"
const secondAssignmentStoryID = "7256b0bb-835c-4166-9600-3d774e919477"

func assignmentDatabase(t *testing.T) (*pgxpool.Pool, *storypostgres.PostgresStoryRepository) {
	t.Helper()
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Delivery')", sprintID, projectID); err != nil {
		t.Fatal(err)
	}
	repo := storypostgres.NewPostgresStoryRepository(pool)
	for _, id := range []string{storyID, secondAssignmentStoryID} {
		story := validStory(projectID)
		story.ID = id
		if err := repo.Create(context.Background(), story); err != nil {
			t.Fatal(err)
		}
	}
	return pool, repo
}

func assertAssignmentCount(t *testing.T, pool *pgxpool.Pool, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("assignment count = %d, want %d", count, want)
	}
}

func TestSprintStoriesMigrationCanBeReversedAndReapplied(t *testing.T) {
	pool, _ := assignmentDatabase(t)
	applyStoryMigration(t, pool, "000008_add_sprint_closed.down.sql")
	applyStoryMigration(t, pool, "000007_create_sprint_stories.down.sql")
	applyStoryMigration(t, pool, "000007_create_sprint_stories.up.sql")
	applyStoryMigration(t, pool, "000008_add_sprint_closed.up.sql")
	var exists, closed bool
	if err := pool.QueryRow(context.Background(), "SELECT to_regclass('sprint_stories') IS NOT NULL, is_closed FROM sprints WHERE id = $1", sprintID).Scan(&exists, &closed); err != nil {
		t.Fatal(err)
	}
	if !exists || closed {
		t.Fatalf("reapplied association/open default = %v/%v", exists, closed)
	}
}

func TestAssignStoriesAtomicallyPreservesBacklogAndStoryFields(t *testing.T) {
	pool, repo := assignmentDatabase(t)
	before, err := repo.ListByProject(context.Background(), projectID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID, secondAssignmentStoryID}); err != nil {
		t.Fatalf("AssignStories() error = %v", err)
	}
	assertAssignmentCount(t, pool, 2)
	// The acceptance contract is the public backlog query, not merely retained rows.
	after, err := repo.ListByProject(context.Background(), projectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 2 || !reflect.DeepEqual(before, after) {
		t.Fatalf("ListByProject after assignment = %#v, want unchanged %#v", after, before)
	}
}

func TestAssignStoriesRejectsMismatchedRouteProjectWithoutWrites(t *testing.T) {
	pool, repo := assignmentDatabase(t)
	insertProject(t, pool, otherProjectID)
	for _, routeProject := range []string{otherProjectID, "ec44c1d7-5572-47d9-ad01-4fa0d79d8f00"} {
		err := repo.AssignStoriesForProject(context.Background(), routeProject, sprintID, []string{storyID})
		if !errors.Is(err, application.ErrProjectMismatch) {
			t.Fatalf("mismatched/missing route project error = %v", err)
		}
		assertAssignmentCount(t, pool, 0)
	}
}

func TestAssignStoriesRejectsCrossProjectBatchWithoutPartialWrites(t *testing.T) {
	pool, repo := assignmentDatabase(t)
	insertProject(t, pool, otherProjectID)
	other := validStory(otherProjectID)
	other.ID = "331a7ff5-12eb-4b2d-83fd-b51fd77e413d"
	if err := repo.Create(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID, other.ID})
	if !errors.Is(err, application.ErrProjectMismatch) {
		t.Fatalf("cross-project error = %v", err)
	}
	assertAssignmentCount(t, pool, 0)
	// Composite foreign keys enforce this even when bypassing the application.
	_, err = pool.Exec(context.Background(), "INSERT INTO sprint_stories (sprint_id, story_id, project_id) VALUES ($1, $2, $3)", sprintID, other.ID, projectID)
	assertDatabaseError(t, err, "23503", "sprint_stories_story_project_fkey")
	assertAssignmentCount(t, pool, 0)
}

func TestAssignStoriesRollsBackWhenAnInsertFails(t *testing.T) {
	pool, repo := assignmentDatabase(t)
	if _, err := pool.Exec(context.Background(), `CREATE FUNCTION reject_second_assignment() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN IF NEW.story_id = '7256b0bb-835c-4166-9600-3d774e919477'::uuid THEN RAISE EXCEPTION 'forced write failure'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER reject_assignment BEFORE INSERT ON sprint_stories FOR EACH ROW EXECUTE FUNCTION reject_second_assignment();`); err != nil {
		t.Fatal(err)
	}
	err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID, secondAssignmentStoryID})
	if err == nil || errors.Is(err, application.ErrStoryAlreadyAssigned) || errors.Is(err, application.ErrStoryNotFound) {
		t.Fatalf("forced insert failure misclassified: %v", err)
	}
	assertAssignmentCount(t, pool, 0)
}

func TestAssignStoriesConcurrentDuplicateHasSingleWinner(t *testing.T) {
	pool, repo := assignmentDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Hold the sprint lock until both competing assignments are observably waiting.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, "SELECT id FROM sprints WHERE id = $1 FOR UPDATE", sprintID); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	for range 2 {
		go func() {
			results <- repo.AssignStoriesForProject(ctx, projectID, sprintID, []string{storyID, secondAssignmentStoryID})
		}()
	}
	waitForAssignmentLocks(t, ctx, pool, 2)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatalf("concurrent results = %v, %v; want one success", first, second)
	}
	loser := first
	if loser == nil {
		loser = second
	}
	if !errors.Is(loser, application.ErrStoryAlreadyAssigned) {
		t.Fatalf("losing assignment error = %v", loser)
	}
	assertAssignmentCount(t, pool, 2)
}

func waitForAssignmentLocks(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want int) {
	t.Helper()
	for {
		var waiting int
		err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database()
			AND query LIKE 'SELECT project_id::text, is_closed FROM sprints%'
			AND wait_event_type = 'Lock'`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting >= want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("assignment did not wait on sprint lock: %v", ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestAssignStoriesWaitsForConcurrentSprintClose(t *testing.T) {
	pool, repo := assignmentDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	closeTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer closeTx.Rollback(context.Background())
	if _, err := closeTx.Exec(ctx, "UPDATE sprints SET is_closed = true WHERE id = $1", sprintID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		result <- repo.AssignStoriesForProject(ctx, projectID, sprintID, []string{storyID, secondAssignmentStoryID})
	}()
	waitForAssignmentLocks(t, ctx, pool, 1)
	if err := closeTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, application.ErrSprintClosed) {
			t.Fatalf("assignment after concurrent close = %v", err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	assertAssignmentCount(t, pool, 0)
}

func TestAssignStoriesRejectsClosedSprintWithoutWrites(t *testing.T) {
	pool, repo := assignmentDatabase(t)
	if _, err := pool.Exec(context.Background(), "UPDATE sprints SET is_closed = true WHERE id = $1", sprintID); err != nil {
		t.Fatal(err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID, secondAssignmentStoryID}); !errors.Is(err, application.ErrSprintClosed) {
		t.Fatalf("closed sprint error = %v", err)
	}
	assertAssignmentCount(t, pool, 0)
}

func TestAssignStoriesRejectsMissingResourcesAndDuplicates(t *testing.T) {
	pool, repo := assignmentDatabase(t)
	missingID := "ec44c1d7-5572-47d9-ad01-4fa0d79d8f00"
	if err := repo.AssignStoriesForProject(context.Background(), projectID, missingID, []string{storyID}); !errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("missing sprint error = %v", err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{missingID}); !errors.Is(err, application.ErrStoryNotFound) {
		t.Errorf("missing story error = %v", err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID, missingID}); !errors.Is(err, application.ErrStoryNotFound) {
		t.Errorf("missing batch error = %v", err)
	}
	useCase := application.NewAssignStoriesUseCase(repo)
	for _, tc := range []struct {
		ids  []string
		want error
	}{
		{nil, application.ErrEmptyStorySelection},
		{[]string{storyID, storyID}, application.ErrDuplicateStoryID},
	} {
		if err := useCase.Execute(context.Background(), application.AssignStoriesCommand{ProjectID: projectID, SprintID: sprintID, StoryIDs: tc.ids}); !errors.Is(err, tc.want) {
			t.Errorf("invalid batch error = %v, want %v", err, tc.want)
		}
	}
	assertAssignmentCount(t, pool, 0)
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{secondAssignmentStoryID, storyID}); !errors.Is(err, application.ErrStoryAlreadyAssigned) {
		t.Errorf("existing association error = %v", err)
	}
	assertAssignmentCount(t, pool, 1)
}
