package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
	defectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/defect/infrastructure/postgres"
)

// defectRecordID is reused for single-row tests; the duplicates test uses a second one.
const (
	defectRecordID2  = "f47ac10b-58cc-4372-a567-0e02b2c3d481"
	defectRecordID3  = "f47ac10b-58cc-4372-a567-0e02b2c3d482"
	unknownSprintID  = "70b06eb1-92e5-4384-8fc1-d18293a4b5c7"
	unknownProjectID = "70b06eb1-92e5-4384-8fc1-d18293a4b5c8"
)

func seededDefectDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := defectDatabaseAtVersion(t, 14)
	seedDefectParents(t, pool)
	return pool
}

func TestDefectRepositoryPersistsDefect(t *testing.T) {
	pool := seededDefectDatabase(t)
	repository := defectpostgres.NewPostgresDefectRepository(pool)

	err := repository.Create(context.Background(), defectof(t, defectRecordID, defectProjectAID, defectStoryID, defectSprintID))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var description, severity, status string
	if err := pool.QueryRow(context.Background(), `
		SELECT description, severity, status FROM defects WHERE id = $1 AND project_id = $2 AND story_id = $3 AND sprint_id = $4
	`, defectRecordID, defectProjectAID, defectStoryID, defectSprintID).Scan(&description, &severity, &status); err != nil {
		t.Fatalf("read persisted defect: %v", err)
	}
	if description != "El login acepta contraseñas vacías" || severity != "alta" || status != "detectado" {
		t.Errorf("persisted defect = (%q, %q, %q), want description/severity/status preserved", description, severity, status)
	}
	if got := defectCount(t, pool); got != 1 {
		t.Errorf("defects rows = %d, want 1", got)
	}
}

func TestDefectRepositoryListsProjectDefectsWithResolutionSprint(t *testing.T) {
	pool := seededDefectDatabase(t)
	repository := defectpostgres.NewPostgresDefectRepository(pool)
	ctx := context.Background()

	if err := repository.Create(ctx, defectof(t, defectRecordID, defectProjectAID, defectStoryID, defectSprintID)); err != nil {
		t.Fatalf("Create open defect: %v", err)
	}
	resolved := defectof(t, defectRecordID2, defectProjectAID, defectBacklogStoryID, defectSprintID)
	resolved.Description = "Resolved defect"
	resolved.Status = "resuelto"
	resolved.ResolutionSprintID = defectSprintID
	if err := repository.Create(ctx, resolved); err != nil {
		t.Fatalf("Create resolved defect: %v", err)
	}
	if err := repository.Create(ctx, defectof(t, defectRecordID3, defectProjectBID, defectStoryBID, defectSprintBID)); err != nil {
		t.Fatalf("Create other-project defect: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE defects SET status = 'resuelto', resolution_sprint_id = $2 WHERE id = $1`, defectRecordID2, defectSprintID); err != nil {
		t.Fatalf("set resolution Sprint: %v", err)
	}

	got, err := repository.ListByProject(ctx, defectProjectAID)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("listed defects = %d, want 2 project A records: %+v", len(got), got)
	}
	if got[0].ID != defectRecordID || got[0].Status != "detectado" || got[0].ResolutionSprintID != "" {
		t.Errorf("open defect = %+v, want original status and no resolution Sprint", got[0])
	}
	if got[1].ID != defectRecordID2 || got[1].Description != "Resolved defect" || got[1].Status != "resuelto" || got[1].ResolutionSprintID != defectSprintID {
		t.Errorf("resolved defect = %+v, want resolution Sprint and stored fields", got[1])
	}
}

func TestDefectRepositoryListsEmptyExistingProject(t *testing.T) {
	pool := seededDefectDatabase(t)
	got, err := defectpostgres.NewPostgresDefectRepository(pool).ListByProject(context.Background(), defectProjectAID)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("listed defects = %#v, want non-nil empty slice", got)
	}
}

func TestDefectRepositoryListsUnknownProjectAsNotFound(t *testing.T) {
	pool := seededDefectDatabase(t)
	_, err := defectpostgres.NewPostgresDefectRepository(pool).ListByProject(context.Background(), unknownProjectID)
	if !errors.Is(err, application.ErrProjectNotFound) {
		t.Fatalf("ListByProject error = %v, want ErrProjectNotFound", err)
	}
}

func TestDefectRepositoryReferencePrecedence(t *testing.T) {
	pool := seededDefectDatabase(t)
	repository := defectpostgres.NewPostgresDefectRepository(pool)
	ctx := context.Background()

	// Precedencia 1: el proyecto de ruta manda (tales UUIDs no existen en ninguna tabla).
	if err := repository.Create(ctx, defectof(t, defectRecordID, unknownProjectID, unknownStoryID, unknownSprintID)); !errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("unknown project error = %v, want ErrProjectNotFound", err)
	}

	// Precedencia 2: una historia ajena reporta story_not_found aunque el Sprint también lo esté.
	if err := repository.Create(ctx, defectof(t, defectRecordID, defectProjectAID, defectStoryBID, defectSprintBID)); !errors.Is(err, application.ErrStoryNotFound) {
		t.Errorf("cross-project story error = %v, want ErrStoryNotFound (precedence over sprint)", err)
	}

	// Precedencia 3: el Sprint ajeno reporta sprint_not_found con la historia válida.
	if err := repository.Create(ctx, defectof(t, defectRecordID, defectProjectAID, defectStoryID, defectSprintBID)); !errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("cross-project sprint error = %v, want ErrSprintNotFound", err)
	}

	// Precedencia 3: Sprint inexistente.
	if err := repository.Create(ctx, defectof(t, defectRecordID, defectProjectAID, defectStoryID, unknownSprintID)); !errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("unknown sprint error = %v, want ErrSprintNotFound", err)
	}

	if got := defectCount(t, pool); got != 0 {
		t.Errorf("defects rows after failed references = %d, want 0", got)
	}
}

func TestDefectRepositoryAcceptsBacklogStory(t *testing.T) {
	// La historia no está asignada a ningún Sprint; el Sprint de detención del proyecto alcanza.
	pool := seededDefectDatabase(t)
	repository := defectpostgres.NewPostgresDefectRepository(pool)

	if err := repository.Create(context.Background(), defectof(t, defectRecordID, defectProjectAID, defectBacklogStoryID, defectSprintID)); err != nil {
		t.Fatalf("Create over backlog story: %v", err)
	}
	if got := defectCount(t, pool); got != 1 {
		t.Errorf("defects rows = %d, want 1", got)
	}
}

func TestDefectRepositoryAcceptsClosedDetectionSprint(t *testing.T) {
	pool := seededDefectDatabase(t)
	closeSprint(t, pool, defectSprintID)
	repository := defectpostgres.NewPostgresDefectRepository(pool)

	if err := repository.Create(context.Background(), defectof(t, defectRecordID, defectProjectAID, defectStoryID, defectSprintID)); err != nil {
		t.Fatalf("Create over closed detection sprint: %v", err)
	}
	if got := defectCount(t, pool); got != 1 {
		t.Errorf("defects rows = %d, want 1", got)
	}
}

func TestDefectRepositoryAllowsExactDuplicates(t *testing.T) {
	pool := seededDefectDatabase(t)
	repository := defectpostgres.NewPostgresDefectRepository(pool)
	ctx := context.Background()

	for i, id := range []string{defectRecordID, defectRecordID2, defectRecordID3} {
		if err := repository.Create(ctx, defectof(t, id, defectProjectAID, defectStoryID, defectSprintID)); err != nil {
			t.Fatalf("duplicate create %d: %v", i+1, err)
		}
	}
	if got := defectCount(t, pool); got != 3 {
		t.Errorf("defects rows after duplicates = %d, want 3", got)
	}
}

func TestDefectRepositoryMapsForeignKeyViolationToDomainError(t *testing.T) {
	// Simula que la historia desapareció entre la verificación y el alta: un trigger
	// BEFORE INSERT la borra y el INSERT termina en violación de FK (23503).
	pool := seededDefectDatabase(t)
	createDeletionTrigger(t, pool)
	repository := defectpostgres.NewPostgresDefectRepository(pool)

	err := repository.Create(context.Background(), defectof(t, defectRecordID, defectProjectAID, defectStoryID, defectSprintID))
	if !errors.Is(err, application.ErrStoryNotFound) {
		t.Errorf("FK violation after verification = %v, want ErrStoryNotFound (not 500 and not raw pg error)", err)
	}
	if got := defectCount(t, pool); got != 0 {
		t.Errorf("defects rows after FK violation = %d, want 0", got)
	}
}

func TestDefectRepositoryUnexpectedStorageErrorPassesThrough(t *testing.T) {
	// Un trigger que lanza una excepción arbitraria deja pasar el error sin traducirlo.
	pool := seededDefectDatabase(t)
	createRaisingTrigger(t, pool)
	repository := defectpostgres.NewPostgresDefectRepository(pool)

	err := repository.Create(context.Background(), defectof(t, defectRecordID, defectProjectAID, defectStoryID, defectSprintID))
	if err == nil {
		t.Fatal("Create: want unexpected storage error")
	}
	if errors.Is(err, application.ErrStoryNotFound) || errors.Is(err, application.ErrProjectNotFound) || errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("unexpected storage error translated to a domain sentinel: %v", err)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Message != "simulated storage failure" {
		t.Errorf("storage error = %v, want the raw trigger failure", err)
	}
}

func TestDefectRepositoryUpdatesResolutionAtomically(t *testing.T) {
	pool := seededDefectDatabase(t)
	ctx := context.Background()
	repository := defectpostgres.NewPostgresDefectRepository(pool)
	insertDefect(t, pool, defectRecordID, defectProjectAID, defectStoryID, defectSprintID)
	resolutionSprintID := "6f9f8a10-0a15-4be4-9470-a63811a12233"
	insertSprint(t, pool, resolutionSprintID, defectProjectAID)

	updated, err := repository.Update(ctx, application.UpdateDefectCommand{
		ProjectID: defectProjectAID, DefectID: defectRecordID, Status: "resuelto", ResolutionSprintID: resolutionSprintID,
	})
	if err != nil {
		t.Fatalf("Update resolved defect: %v", err)
	}
	if updated.Status != "resuelto" || updated.ResolutionSprintID != resolutionSprintID {
		t.Fatalf("updated defect = %+v, want resolved in %s", updated, resolutionSprintID)
	}

	if _, err := repository.Update(ctx, application.UpdateDefectCommand{
		ProjectID: defectProjectAID, DefectID: defectRecordID, Status: "resuelto", ResolutionSprintID: defectSprintBID,
	}); !errors.Is(err, application.ErrResolutionSprintNotFound) {
		t.Fatalf("cross-project resolution Sprint error = %v, want ErrResolutionSprintNotFound", err)
	}
	assertStoredDefectState(t, pool, defectRecordID, "resuelto", resolutionSprintID)

	if _, err := repository.Update(ctx, application.UpdateDefectCommand{
		ProjectID: defectProjectAID, DefectID: "70b06eb1-92e5-4384-8fc1-d18293a4b5c9", Status: "detectado",
	}); !errors.Is(err, application.ErrDefectNotFound) {
		t.Fatalf("missing defect error = %v, want ErrDefectNotFound", err)
	}

	if _, err := repository.Update(ctx, application.UpdateDefectCommand{
		ProjectID: defectProjectAID, DefectID: defectRecordID, Status: "en_correccion",
	}); err != nil {
		t.Fatalf("reopen defect: %v", err)
	}
	assertStoredDefectState(t, pool, defectRecordID, "en_correccion", "")
}

func TestDefectRepositoryRejectsResolutionSprintBeforeDetection(t *testing.T) {
	pool := seededDefectDatabase(t)
	earlierSprintID := "6f9f8a10-0a15-4be4-9470-a63811a12234"
	detectionSprintID := "6f9f8a10-0a15-4be4-9470-a63811a12235"
	insertSprint(t, pool, earlierSprintID, defectProjectAID)
	insertSprint(t, pool, detectionSprintID, defectProjectAID)
	insertDefect(t, pool, defectRecordID, defectProjectAID, defectStoryID, detectionSprintID)
	repository := defectpostgres.NewPostgresDefectRepository(pool)

	if _, err := repository.Update(context.Background(), application.UpdateDefectCommand{
		ProjectID: defectProjectAID, DefectID: defectRecordID, Status: "resuelto", ResolutionSprintID: earlierSprintID,
	}); !errors.Is(err, application.ErrResolutionSprintBeforeDetection) {
		t.Fatalf("earlier resolution Sprint error = %v, want ErrResolutionSprintBeforeDetection", err)
	}
	assertStoredDefectState(t, pool, defectRecordID, "detectado", "")
}

func assertStoredDefectState(t *testing.T, pool *pgxpool.Pool, id, wantStatus, wantResolutionSprint string) {
	t.Helper()
	var status string
	var resolutionSprint *string
	if err := pool.QueryRow(context.Background(), "SELECT status, resolution_sprint_id::text FROM defects WHERE id = $1", id).Scan(&status, &resolutionSprint); err != nil {
		t.Fatalf("read defect state: %v", err)
	}
	gotResolutionSprint := ""
	if resolutionSprint != nil {
		gotResolutionSprint = *resolutionSprint
	}
	if status != wantStatus || gotResolutionSprint != wantResolutionSprint {
		t.Errorf("stored state = (%q, %q), want (%q, %q)", status, gotResolutionSprint, wantStatus, wantResolutionSprint)
	}
}

func defectof(t *testing.T, id, projectID, storyID, sprintID string) domain.Defect {
	t.Helper()
	defect, err := domain.NewDefect(domain.NewDefectParams{
		ID:          id,
		ProjectID:   projectID,
		StoryID:     storyID,
		SprintID:    sprintID,
		Description: "El login acepta contraseñas vacías",
		Severity:    "alta",
		Status:      "detectado",
	})
	if err != nil {
		t.Fatalf("build domain defect: %v", err)
	}
	return defect
}

func closeSprint(t *testing.T, pool *pgxpool.Pool, sprintID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "UPDATE sprints SET is_closed = true WHERE id = $1", sprintID); err != nil {
		t.Fatalf("close sprint: %v", err)
	}
}

// createDeletionTrigger borra la historia referenciada antes de cada INSERT en defects.
func createDeletionTrigger(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		CREATE OR REPLACE FUNCTION defect_delete_story() RETURNS trigger AS $$
		BEGIN
			DELETE FROM stories WHERE id = NEW.story_id AND project_id = NEW.project_id;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER defect_delete_story_trigger
			BEFORE INSERT ON defects
			FOR EACH ROW EXECUTE FUNCTION defect_delete_story();
	`)
	if err != nil {
		t.Fatalf("create deletion trigger: %v", err)
	}
}

// createRaisingTrigger interrumpe el INSERT con una excepción arbitraria.
func createRaisingTrigger(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		CREATE OR REPLACE FUNCTION defect_raise_storage() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'simulated storage failure';
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER defect_raise_storage_trigger
			BEFORE INSERT ON defects
			FOR EACH ROW EXECUTE FUNCTION defect_raise_storage();
	`)
	if err != nil {
		t.Fatalf("create raising trigger: %v", err)
	}
}
