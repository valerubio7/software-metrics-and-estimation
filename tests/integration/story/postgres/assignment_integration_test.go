package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
)

const sprintID = "b2a6a455-06e2-41ee-b011-7462c71375a0"

func TestSprintStoriesMigrationCanBeReversedAndReapplied(t *testing.T) {
	pool := storyDatabase(t)
	applyStoryMigration(t, pool, "000005_create_sprint_stories.down.sql")
	applyStoryMigration(t, pool, "000005_create_sprint_stories.up.sql")
	var exists bool
	if err := pool.QueryRow(context.Background(), "SELECT to_regclass('sprint_stories') IS NOT NULL").Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("sprint_stories table missing after reapplying migration")
	}
}

func TestAssignStoriesAtomicallyPreservesBacklogAndStoryFields(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	_, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Entrega')", sprintID, projectID)
	if err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	first, second := validStory(projectID), validStory(projectID)
	second.ID = "7256b0bb-835c-4166-9600-3d774e919477"
	repo := storypostgres.NewPostgresStoryRepository(pool)
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{first.ID, second.ID}); err != nil {
		t.Fatalf("AssignStories() error = %v", err)
	}
	var linked, backlog, unchanged int
	err = pool.QueryRow(context.Background(), `SELECT count(*), count(*) FILTER (WHERE s.project_id = $2), count(*) FILTER (WHERE s.status = 'pendiente' AND s.title = '  Registro  ') FROM sprint_stories ss JOIN stories s ON s.id = ss.story_id WHERE ss.sprint_id = $1`, sprintID, projectID).Scan(&linked, &backlog, &unchanged)
	if err != nil {
		t.Fatal(err)
	}
	if linked != 2 || backlog != 2 || unchanged != 2 {
		t.Fatalf("linked/backlog/unchanged = %d/%d/%d, want 2/2/2", linked, backlog, unchanged)
	}
}

func TestAssignStoriesRejectsMismatchedRouteProjectWithoutWrites(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	insertProject(t, pool, otherProjectID)
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Entrega')", sprintID, projectID); err != nil {
		t.Fatal(err)
	}
	if err := storypostgres.NewPostgresStoryRepository(pool).Create(context.Background(), validStory(projectID)); err != nil {
		t.Fatal(err)
	}
	err := storypostgres.NewPostgresStoryRepository(pool).AssignStoriesForProject(context.Background(), otherProjectID, sprintID, []string{storyID})
	if !errors.Is(err, application.ErrProjectMismatch) {
		t.Fatalf("AssignStories() error = %v, want ErrProjectMismatch", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("assignments for mismatched route project = %d, want 0", count)
	}
}

func TestAssignStoriesRejectsCrossProjectBatchWithoutPartialWrites(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	insertProject(t, pool, otherProjectID)
	_, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Entrega')", sprintID, projectID)
	if err != nil {
		t.Fatal(err)
	}
	first, other := validStory(projectID), validStory(otherProjectID)
	other.ID = "7256b0bb-835c-4166-9600-3d774e919477"
	repo := storypostgres.NewPostgresStoryRepository(pool)
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	err = repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{first.ID, other.ID})
	if !errors.Is(err, application.ErrProjectMismatch) {
		t.Fatalf("AssignStories() error = %v, want ErrProjectMismatch", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partial assignments = %d, want 0", count)
	}
}

func TestAssignStoriesRollsBackWhenAnInsertFails(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Entrega')", sprintID, projectID); err != nil {
		t.Fatal(err)
	}
	first, second := validStory(projectID), validStory(projectID)
	second.ID = "7256b0bb-835c-4166-9600-3d774e919477"
	repo := storypostgres.NewPostgresStoryRepository(pool)
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `CREATE FUNCTION reject_second_assignment() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN IF NEW.story_id = '7256b0bb-835c-4166-9600-3d774e919477'::uuid THEN RAISE EXCEPTION 'forced write failure'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER reject_assignment BEFORE INSERT ON sprint_stories FOR EACH ROW EXECUTE FUNCTION reject_second_assignment();`); err != nil {
		t.Fatal(err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{first.ID, second.ID}); err == nil {
		t.Fatal("AssignStories() succeeded despite forced insert error")
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partial assignments after failed insert = %d, want 0", count)
	}
}

func TestAssignStoriesConcurrentDuplicateHasSingleWinner(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Entrega')", sprintID, projectID); err != nil {
		t.Fatal(err)
	}
	if err := storypostgres.NewPostgresStoryRepository(pool).Create(context.Background(), validStory(projectID)); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- storypostgres.NewPostgresStoryRepository(pool).AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID})
		}()
	}
	close(start)
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
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("assignment count = %d, want 1", count)
	}
}

func TestAssignStoriesWaitsForConcurrentSprintClose(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Open')", sprintID, projectID); err != nil {
		t.Fatal(err)
	}
	if err := storypostgres.NewPostgresStoryRepository(pool).Create(context.Background(), validStory(projectID)); err != nil {
		t.Fatal(err)
	}

	closeTx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer closeTx.Rollback(context.Background())
	if _, err := closeTx.Exec(context.Background(), "UPDATE sprints SET is_closed = true WHERE id = $1", sprintID); err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	go func() {
		result <- storypostgres.NewPostgresStoryRepository(pool).AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID})
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		err := pool.QueryRow(context.Background(), `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE datname = current_database()
			  AND query LIKE 'SELECT project_id::text, is_closed FROM sprints%'
			  AND wait_event_type = 'Lock'
		)`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("assignment did not wait for the concurrent sprint close")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := closeTx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, application.ErrSprintClosed) {
			t.Fatalf("AssignStories() error = %v, want ErrSprintClosed after close commits", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("AssignStories() did not finish after close committed")
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("assignments after concurrent close = %d, want 0", count)
	}
}

func TestAssignStoriesRejectsClosedSprintWithoutWrites(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	if _, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal, is_closed) VALUES ($1, $2, 'Closed', true)", sprintID, projectID); err != nil {
		t.Fatal(err)
	}
	if err := storypostgres.NewPostgresStoryRepository(pool).Create(context.Background(), validStory(projectID)); err != nil {
		t.Fatal(err)
	}
	err := storypostgres.NewPostgresStoryRepository(pool).AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID})
	if !errors.Is(err, application.ErrSprintClosed) {
		t.Fatalf("AssignStories() error = %v, want ErrSprintClosed", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("closed sprint assignments = %d, want 0", count)
	}
}

func TestAssignStoriesRejectsMissingResourcesAndDuplicates(t *testing.T) {
	pool := storyDatabase(t)
	insertProject(t, pool, projectID)
	_, err := pool.Exec(context.Background(), "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Entrega')", sprintID, projectID)
	if err != nil {
		t.Fatal(err)
	}
	repo := storypostgres.NewPostgresStoryRepository(pool)
	if err := repo.AssignStoriesForProject(context.Background(), projectID, "ec44c1d7-5572-47d9-ad01-4fa0d79d8f00", []string{storyID}); !errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("missing sprint error = %v", err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{storyID}); !errors.Is(err, application.ErrStoryNotFound) {
		t.Errorf("missing story error = %v", err)
	}
	story := validStory(projectID)
	eligible := story
	eligible.ID = "7256b0bb-835c-4166-9600-3d774e919477"
	if err := repo.Create(context.Background(), eligible); err != nil {
		t.Fatal(err)
	}
	missingBatchErr := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{eligible.ID, "ec44c1d7-5572-47d9-ad01-4fa0d79d8f00"})
	if !errors.Is(missingBatchErr, application.ErrStoryNotFound) {
		t.Errorf("missing batch error = %v", missingBatchErr)
	}
	var missingBatchCount int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&missingBatchCount); err != nil {
		t.Fatal(err)
	}
	if missingBatchCount != 0 {
		t.Errorf("missing batch assignments = %d, want 0", missingBatchCount)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{eligible.ID}); err != nil {
		t.Fatal(err)
	}
	other := validStory(projectID)
	other.ID = "331a7ff5-12eb-4b2d-83fd-b51fd77e413d"
	if err := repo.Create(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	if err := repo.AssignStoriesForProject(context.Background(), projectID, sprintID, []string{other.ID, eligible.ID}); !errors.Is(err, application.ErrStoryAlreadyAssigned) {
		t.Errorf("existing association error = %v", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("assignment count = %d, want existing one only", count)
	}
}
