package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/domain"
	effortpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/effort/infrastructure/postgres"
)

// Second story of project A (assigned to the Sprint, with its own task), a backlog story of
// project A (no Sprint, no tasks) and a complete second project B. They let the tests prove
// that references are scoped by project and by story.
const (
	effortStoryA2ID  = "0f3f9d4a-2b7e-4c1d-9e5a-6a1b2c3d4e5f"
	effortTaskA2ID   = "1a4a0e5b-3c8f-4d2e-8f6b-7b2c3d4e5f60"
	effortBacklogID  = "2b5b1f6c-4d90-4e3f-9a7c-8c3d4e5f6071"
	effortProjectBID = "3c6c2a7d-5ea1-4f40-8b8d-9d4e5f607182"
	effortSprintBID  = "4d7d3b8e-6fb2-4051-9c9e-ae5f60718293"
	effortStoryBID   = "5e8e4c9f-70c3-4162-8daf-bf60718293a4"
	effortMemberBID  = "6f9f5da0-81d4-4273-9eb0-c07182930a4b"
	effortTaskBID    = "92c280d3-b407-45a6-8be3-f3a4b5c6d7e8"
	effortUnknownID  = "70a06eb1-92e5-4384-8fc1-d18293a4b5c6"
	effortRecordID3  = "81b17fc2-a3f6-4495-9ad2-e293a4b5c6d7"
	effortRecordID4  = "a3d391e4-c518-46b7-9cf4-04b5c6d7e8f9"

	effortStoredWorkDate = "2026-03-10"
)

func TestEffortRepositoryPersistsRecordOverStory(t *testing.T) {
	pool := seededEffortDatabase(t)
	repository := effortpostgres.NewPostgresEffortRepository(pool)

	if err := repository.Create(context.Background(), effortRecord(effortRecordID, nil)); err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	got := storedEffortRecord(t, pool, effortRecordID)
	want := storedEffort{
		ProjectID: effortProjectAID, StoryID: effortStoryID, MemberID: effortMemberID,
		WorkDate: effortStoredWorkDate, Description: "Revision de codigo", Hours: "2.50", HasCreatedAt: true,
	}
	if got != want {
		t.Errorf("stored record = %+v, want %+v", got, want)
	}
	if got.TaskID != nil {
		t.Errorf("stored task_id = %q, want NULL for effort over the story", *got.TaskID)
	}
}

func TestEffortRepositoryPersistsRecordOverTask(t *testing.T) {
	pool := seededEffortDatabase(t)
	repository := effortpostgres.NewPostgresEffortRepository(pool)
	taskID := effortTaskID

	if err := repository.Create(context.Background(), effortRecord(effortRecordID, &taskID)); err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	got := storedEffortRecord(t, pool, effortRecordID)
	if got.TaskID == nil || *got.TaskID != effortTaskID {
		t.Fatalf("stored task_id = %v, want %q", got.TaskID, effortTaskID)
	}
	if got.StoryID != effortStoryID || got.ProjectID != effortProjectAID || got.MemberID != effortMemberID {
		t.Errorf("stored scope = (%s, %s, %s), want (%s, %s, %s)", got.ProjectID, got.StoryID, got.MemberID, effortProjectAID, effortStoryID, effortMemberID)
	}
}

func TestEffortRepositoryCountsTaskEffortOnceForItsStory(t *testing.T) {
	pool := seededEffortDatabase(t)
	repository := effortpostgres.NewPostgresEffortRepository(pool)
	taskID := effortTaskID

	if err := repository.Create(context.Background(), effortRecord(effortRecordID, nil)); err != nil {
		t.Fatalf("Create() over story error = %v", err)
	}
	if err := repository.Create(context.Background(), effortRecord(effortRecordID2, &taskID)); err != nil {
		t.Fatalf("Create() over task error = %v", err)
	}

	var rows int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM effort_records WHERE story_id = $1", effortStoryID).Scan(&rows); err != nil {
		t.Fatalf("count story effort: %v", err)
	}
	if rows != 2 {
		t.Errorf("effort rows of the story = %d, want exactly 2 (one per record, none duplicated for the task)", rows)
	}
}

func TestEffortRepositoryRejectsMissingOrForeignReferencesWithoutWrites(t *testing.T) {
	pool := seededEffortDatabase(t)
	repository := effortpostgres.NewPostgresEffortRepository(pool)

	tests := []struct {
		name   string
		mutate func(*domain.EffortRecord)
		want   error
	}{
		{name: "missing project", mutate: func(r *domain.EffortRecord) { r.ProjectID = effortUnknownID }, want: application.ErrProjectNotFound},
		{name: "missing story", mutate: func(r *domain.EffortRecord) { r.StoryID = effortUnknownID }, want: application.ErrStoryNotFound},
		{name: "story of another project", mutate: func(r *domain.EffortRecord) { r.StoryID = effortStoryBID }, want: application.ErrStoryNotFound},
		{name: "missing member", mutate: func(r *domain.EffortRecord) { r.MemberID = effortUnknownID }, want: application.ErrMemberNotFound},
		{name: "member of another project", mutate: func(r *domain.EffortRecord) { r.MemberID = effortMemberBID }, want: application.ErrMemberNotFound},
		{name: "missing task", mutate: func(r *domain.EffortRecord) { r.TaskID = ptr(effortUnknownID) }, want: application.ErrTaskNotFound},
		{name: "task of another story of the same project", mutate: func(r *domain.EffortRecord) { r.TaskID = ptr(effortTaskA2ID) }, want: application.ErrTaskNotFound},
		{name: "task of another project", mutate: func(r *domain.EffortRecord) { r.TaskID = ptr(effortTaskBID) }, want: application.ErrTaskNotFound},
		{
			name: "everything missing reports the project first",
			mutate: func(r *domain.EffortRecord) {
				r.ProjectID, r.StoryID, r.MemberID, r.TaskID = effortUnknownID, effortUnknownID, effortUnknownID, ptr(effortUnknownID)
			},
			want: application.ErrProjectNotFound,
		},
		{
			name: "story, member and task missing reports the story first",
			mutate: func(r *domain.EffortRecord) {
				r.StoryID, r.MemberID, r.TaskID = effortUnknownID, effortUnknownID, ptr(effortUnknownID)
			},
			want: application.ErrStoryNotFound,
		},
		{
			name: "member and task missing reports the member first",
			mutate: func(r *domain.EffortRecord) {
				r.MemberID, r.TaskID = effortUnknownID, ptr(effortUnknownID)
			},
			want: application.ErrMemberNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := effortRecord(effortRecordID, nil)
			test.mutate(&record)

			err := repository.Create(context.Background(), record)

			if !errors.Is(err, test.want) {
				t.Errorf("Create() error = %v, want %v", err, test.want)
			}
			if got := effortRecordCount(t, pool); got != 0 {
				t.Errorf("effort_records rows after rejected Create() = %d, want 0", got)
			}
		})
	}
}

func TestEffortRepositoryAcceptsClosedSprintAndBacklogStory(t *testing.T) {
	pool := seededEffortDatabase(t)
	repository := effortpostgres.NewPostgresEffortRepository(pool)
	if _, err := pool.Exec(context.Background(), "UPDATE sprints SET is_closed = true WHERE id = $1", effortSprintID); err != nil {
		t.Fatalf("close sprint: %v", err)
	}
	taskID := effortTaskID
	backlogRecord := effortRecord(effortRecordID3, nil)
	backlogRecord.StoryID = effortBacklogID

	tests := []struct {
		name   string
		record domain.EffortRecord
	}{
		{name: "story of a closed sprint", record: effortRecord(effortRecordID, nil)},
		{name: "task of a closed sprint", record: effortRecord(effortRecordID2, &taskID)},
		{name: "story outside any sprint", record: backlogRecord},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := repository.Create(context.Background(), test.record); err != nil {
				t.Fatalf("Create() error = %v, want nil", err)
			}
			if got := storedEffortRecord(t, pool, test.record.ID); got.StoryID != test.record.StoryID {
				t.Errorf("stored story_id = %s, want %s", got.StoryID, test.record.StoryID)
			}
		})
	}
	if got := effortRecordCount(t, pool); got != len(tests) {
		t.Errorf("effort_records rows = %d, want %d", got, len(tests))
	}
}

func TestEffortRepositoryAllowsExactDuplicates(t *testing.T) {
	pool := seededEffortDatabase(t)
	repository := effortpostgres.NewPostgresEffortRepository(pool)

	for _, id := range []string{effortRecordID, effortRecordID2} {
		if err := repository.Create(context.Background(), effortRecord(id, nil)); err != nil {
			t.Fatalf("Create(%s) error = %v, want nil", id, err)
		}
	}

	first, second := storedEffortRecord(t, pool, effortRecordID), storedEffortRecord(t, pool, effortRecordID2)
	if first != second {
		t.Errorf("duplicate rows differ: %+v vs %+v", first, second)
	}
	if got := effortRecordCount(t, pool); got != 2 {
		t.Errorf("effort_records rows = %d, want 2 distinct rows with the same content", got)
	}
}

func TestEffortRepositoryDoesNotCapDailyHours(t *testing.T) {
	pool := seededEffortDatabase(t)
	repository := effortpostgres.NewPostgresEffortRepository(pool)
	ids := []string{effortRecordID, effortRecordID2, effortRecordID3, effortRecordID4}

	for _, id := range ids {
		record := effortRecord(id, nil)
		record.Hours = 8
		if err := repository.Create(context.Background(), record); err != nil {
			t.Fatalf("Create(%s) error = %v, want nil: the daily total must not be capped", id, err)
		}
	}

	var total string
	if err := pool.QueryRow(context.Background(), "SELECT sum(hours)::text FROM effort_records WHERE member_id = $1 AND work_date = '2026-03-10'", effortMemberID).Scan(&total); err != nil {
		t.Fatalf("sum daily hours: %v", err)
	}
	if total != "32.00" {
		t.Errorf("hours registered by the member that day = %s, want 32.00 (24 h already registered plus 8 h)", total)
	}
}

func TestEffortRecordsTableRejectsCrossProjectReferences(t *testing.T) {
	pool := seededEffortDatabase(t)

	tests := []struct {
		name       string
		memberID   string
		storyID    string
		taskID     *string
		constraint string
	}{
		{name: "member of another project", memberID: effortMemberBID, storyID: effortStoryID, constraint: effortMemberProjectFKey},
		{name: "story of another project", memberID: effortMemberID, storyID: effortStoryBID, constraint: effortStoryProjectFKey},
		{name: "task of another story", memberID: effortMemberID, storyID: effortStoryID, taskID: ptr(effortTaskA2ID), constraint: effortTaskStoryProjFKey},
		{name: "task of another project", memberID: effortMemberID, storyID: effortStoryID, taskID: ptr(effortTaskBID), constraint: effortTaskStoryProjFKey},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rawInsertEffort(pool, effortRecordID, effortProjectAID, test.storyID, test.taskID, test.memberID, 2.5, "Revision de codigo")

			assertDatabaseError(t, err, "23503", test.constraint)
		})
	}
	if got := effortRecordCount(t, pool); got != 0 {
		t.Errorf("effort_records rows after rejected inserts = %d, want 0", got)
	}
}

func TestEffortRecordsTableSkipsTaskConstraintWhenTaskIsNull(t *testing.T) {
	pool := seededEffortDatabase(t)

	if err := rawInsertEffort(pool, effortRecordID, effortProjectAID, effortStoryID, nil, effortMemberID, 2.5, "Sin tarea"); err != nil {
		t.Fatalf("insert with NULL task_id error = %v, want nil (MATCH SIMPLE skips the task foreign key)", err)
	}

	if got := storedEffortRecord(t, pool, effortRecordID); got.TaskID != nil {
		t.Errorf("stored task_id = %q, want NULL", *got.TaskID)
	}
	err := rawInsertEffort(pool, effortRecordID2, effortProjectAID, effortStoryID, ptr(effortUnknownID), effortMemberID, 2.5, "Tarea inexistente")
	assertDatabaseError(t, err, "23503", effortTaskStoryProjFKey)
}

func TestEffortRecordsTableEnforcesChecks(t *testing.T) {
	pool := seededEffortDatabase(t)
	multibyte500 := strings.Repeat("ñ", 500)

	tests := []struct {
		name        string
		hours       float64
		description string
		constraint  string // empty means the row is accepted
	}{
		{name: "zero hours", hours: 0, description: "ok", constraint: effortHoursRangeCheck},
		{name: "hours above 24", hours: 24.01, description: "ok", constraint: effortHoursRangeCheck},
		{name: "empty description", hours: 1, description: "", constraint: effortDescriptionLenCheck},
		{name: "description of 501 characters", hours: 1, description: strings.Repeat("a", 501), constraint: effortDescriptionLenCheck},
		{name: "description of 501 multibyte characters", hours: 1, description: multibyte500 + "ñ", constraint: effortDescriptionLenCheck},
		{name: "hours of exactly 24", hours: 24, description: "ok"},
		{name: "hours of 0.01", hours: 0.01, description: "ok"},
		{name: "description of 500 multibyte characters", hours: 1, description: multibyte500},
	}

	accepted := 0
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id := fmt.Sprintf("00000000-0000-4000-8000-%012d", index+1)
			err := rawInsertEffort(pool, id, effortProjectAID, effortStoryID, nil, effortMemberID, test.hours, test.description)

			if test.constraint != "" {
				assertDatabaseError(t, err, "23514", test.constraint)
				return
			}
			if err != nil {
				t.Fatalf("insert error = %v, want nil", err)
			}
			accepted++
		})
	}
	if got := effortRecordCount(t, pool); got != accepted || accepted != 3 {
		t.Errorf("effort_records rows = %d (accepted %d), want 3 accepted rows and no row from the rejected ones", got, accepted)
	}
}

func TestEffortRecordsRestrictDeletesOfReferencedParents(t *testing.T) {
	pool := seededEffortDatabase(t)
	repository := effortpostgres.NewPostgresEffortRepository(pool)
	taskID := effortTaskID
	backlogRecord := effortRecord(effortRecordID3, nil)
	backlogRecord.StoryID = effortBacklogID
	for _, record := range []domain.EffortRecord{effortRecord(effortRecordID, &taskID), backlogRecord} {
		if err := repository.Create(context.Background(), record); err != nil {
			t.Fatalf("Create(%s) error = %v", record.ID, err)
		}
	}

	tests := []struct {
		name       string
		statement  string
		argument   string
		constraint string
	}{
		{name: "task with effort", statement: "DELETE FROM tasks WHERE id = $1", argument: effortTaskID, constraint: effortTaskStoryProjFKey},
		{name: "member with effort", statement: "DELETE FROM project_members WHERE id = $1", argument: effortMemberID, constraint: effortMemberProjectFKey},
		{name: "story with effort", statement: "DELETE FROM stories WHERE id = $1", argument: effortBacklogID, constraint: effortStoryProjectFKey},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := pool.Exec(context.Background(), test.statement, test.argument)

			assertDatabaseError(t, err, "23503", test.constraint)
		})
	}
	if got := effortRecordCount(t, pool); got != 2 {
		t.Errorf("effort_records rows after restricted deletes = %d, want 2", got)
	}
}

// rawInsertEffort inserts a row straight into effort_records, bypassing the repository, and
// returns the database error so tests can assert the exact constraint that rejected it.
func rawInsertEffort(pool *pgxpool.Pool, id, projectID, storyID string, taskID *string, memberID string, hours float64, description string) error {
	_, err := pool.Exec(context.Background(), `
		INSERT INTO effort_records (id, project_id, story_id, task_id, member_id, work_date, description, hours)
		VALUES ($1, $2, $3, $4, $5, '2026-03-10', $6, $7)
	`, id, projectID, storyID, taskID, memberID, description, hours)
	return err
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

// storedEffort is a text rendering of one effort_records row, comparable with ==.
type storedEffort struct {
	ProjectID, StoryID, MemberID string
	TaskID                       *string
	WorkDate, Description, Hours string
	HasCreatedAt                 bool
}

func storedEffortRecord(t *testing.T, pool *pgxpool.Pool, id string) storedEffort {
	t.Helper()
	var stored storedEffort
	err := pool.QueryRow(context.Background(), `
		SELECT project_id::text, story_id::text, task_id::text, member_id::text,
		       work_date::text, description, hours::text, created_at IS NOT NULL
		FROM effort_records WHERE id = $1
	`, id).Scan(&stored.ProjectID, &stored.StoryID, &stored.TaskID, &stored.MemberID,
		&stored.WorkDate, &stored.Description, &stored.Hours, &stored.HasCreatedAt)
	if err != nil {
		t.Fatalf("read effort record %s: %v", id, err)
	}
	return stored
}

// effortRecord returns a valid record of the seeded member over the seeded story of project A.
func effortRecord(id string, taskID *string) domain.EffortRecord {
	return domain.EffortRecord{
		ID:          id,
		ProjectID:   effortProjectAID,
		StoryID:     effortStoryID,
		TaskID:      taskID,
		MemberID:    effortMemberID,
		WorkDate:    time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC),
		Description: "Revision de codigo",
		Hours:       2.5,
	}
}

func ptr[T any](value T) *T { return &value }

// seededEffortDatabase returns a migrated database holding project A (Sprint, two stories with
// one task each, a backlog story and a member) and an unrelated project B with the same shape.
func seededEffortDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := effortDatabase(t)
	seedEffortParents(t, pool)
	insertStory(t, pool, effortStoryA2ID, effortProjectAID)
	assignStory(t, pool, effortSprintID, effortStoryA2ID, effortProjectAID)
	insertTask(t, pool, effortTaskA2ID, effortProjectAID, effortSprintID, effortStoryA2ID)
	insertStory(t, pool, effortBacklogID, effortProjectAID)

	insertProject(t, pool, effortProjectBID)
	insertSprint(t, pool, effortSprintBID, effortProjectBID)
	insertStory(t, pool, effortStoryBID, effortProjectBID)
	assignStory(t, pool, effortSprintBID, effortStoryBID, effortProjectBID)
	insertMember(t, pool, effortMemberBID, effortProjectBID, "Beto Gomez")
	insertTask(t, pool, effortTaskBID, effortProjectBID, effortSprintBID, effortStoryBID)
	return pool
}
