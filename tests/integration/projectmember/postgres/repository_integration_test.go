package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
	memberpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/infrastructure/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/tests/integration/testpostgres"
)

func TestRegisterMembersPersistsBatchAndRejectsDuplicate(t *testing.T) {
	pool := newPostgresPool(t)
	applyMigrations(t, pool)
	projectID := createProject(t, pool)
	repo := memberpostgres.NewPostgresMemberRepository(pool)
	var beforeName, beforeStartDate, beforeFinishDate string
	if err := pool.QueryRow(context.Background(), `SELECT name, start_date::text, planned_finish_date::text FROM projects WHERE id=$1`, projectID).Scan(&beforeName, &beforeStartDate, &beforeFinishDate); err != nil {
		t.Fatal(err)
	}
	members := []domain.Member{{ID: "00000000-0000-0000-0000-000000000101", ProjectID: projectID, FullName: "Alex Example", Email: stringPointer("alex@example.com")}, {ID: "00000000-0000-0000-0000-000000000102", ProjectID: projectID, FullName: "Sam Example"}}
	if err := repo.Register(context.Background(), projectID, members); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM project_members WHERE project_id=$1`, projectID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("persisted count = %d, want 2", count)
	}
	var afterName, afterStartDate, afterFinishDate string
	if err := pool.QueryRow(context.Background(), `SELECT name, start_date::text, planned_finish_date::text FROM projects WHERE id=$1`, projectID).Scan(&afterName, &afterStartDate, &afterFinishDate); err != nil {
		t.Fatal(err)
	}
	if beforeName != afterName || beforeStartDate != afterStartDate || beforeFinishDate != afterFinishDate {
		t.Fatalf("project fields changed after registration: before=(%q, %q, %q), after=(%q, %q, %q)", beforeName, beforeStartDate, beforeFinishDate, afterName, afterStartDate, afterFinishDate)
	}
	if err := repo.Register(context.Background(), projectID, members[:1]); !errors.Is(err, application.ErrDuplicateMember) {
		t.Fatalf("duplicate Register() error = %v, want ErrDuplicateMember", err)
	}
}

func TestRegisterMembersMissingProjectAndNullEmailDuplicate(t *testing.T) {
	pool := newPostgresPool(t)
	applyMigrations(t, pool)
	repo := memberpostgres.NewPostgresMemberRepository(pool)
	missing := "00000000-0000-0000-0000-000000000099"
	if err := repo.Register(context.Background(), missing, []domain.Member{{ID: "00000000-0000-0000-0000-000000000103", ProjectID: missing, FullName: "Absent"}}); !errors.Is(err, application.ErrProjectNotFound) {
		t.Fatalf("missing project error = %v", err)
	}
	projectID := createProject(t, pool)
	member := domain.Member{ID: "00000000-0000-0000-0000-000000000104", ProjectID: projectID, FullName: "No Email"}
	if err := repo.Register(context.Background(), projectID, []domain.Member{member}); err != nil {
		t.Fatal(err)
	}
	member.ID = "00000000-0000-0000-0000-000000000105"
	if err := repo.Register(context.Background(), projectID, []domain.Member{member}); !errors.Is(err, application.ErrDuplicateMember) {
		t.Fatalf("NULL duplicate error = %v, want ErrDuplicateMember", err)
	}
}

func TestRegisterMembersInsertionFailureRollsBackWholeBatch(t *testing.T) {
	pool := newPostgresPool(t)
	applyMigrations(t, pool)
	projectID := createProject(t, pool)
	_, err := pool.Exec(context.Background(), `
		CREATE FUNCTION reject_trigger_failure_member() RETURNS trigger AS $$
		BEGIN
			IF NEW.full_name = 'Trigger Failure' THEN
				RAISE EXCEPTION 'injected member insertion failure';
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER reject_trigger_failure_member BEFORE INSERT ON project_members
		FOR EACH ROW EXECUTE FUNCTION reject_trigger_failure_member();
	`)
	if err != nil {
		t.Fatal(err)
	}

	repo := memberpostgres.NewPostgresMemberRepository(pool)
	members := []domain.Member{
		{ID: "00000000-0000-0000-0000-000000000110", ProjectID: projectID, FullName: "Valid First"},
		{ID: "00000000-0000-0000-0000-000000000111", ProjectID: projectID, FullName: "Trigger Failure"},
	}
	err = repo.Register(context.Background(), projectID, members)
	if err == nil {
		t.Fatal("Register() error = nil, want injected database failure")
	}
	if errors.Is(err, application.ErrDuplicateMember) || !strings.Contains(err.Error(), "injected member insertion failure") {
		t.Fatalf("Register() error = %v, want unwrapped injected database failure", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM project_members WHERE project_id=$1`, projectID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("persisted members after insertion failure = %d, want 0", count)
	}
}

func TestRegisterMembersBatchRollbackAndConcurrentDuplicate(t *testing.T) {
	pool := newPostgresPool(t)
	applyMigrations(t, pool)
	projectID := createProject(t, pool)
	repo := memberpostgres.NewPostgresMemberRepository(pool)
	valid := domain.Member{ID: "00000000-0000-0000-0000-000000000106", ProjectID: projectID, FullName: "First"}
	duplicate := domain.Member{ID: "00000000-0000-0000-0000-000000000107", ProjectID: projectID, FullName: "Existing"}
	if err := repo.Register(context.Background(), projectID, []domain.Member{duplicate}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Register(context.Background(), projectID, []domain.Member{valid, duplicate}); !errors.Is(err, application.ErrDuplicateMember) {
		t.Fatalf("atomic duplicate error = %v", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM project_members WHERE project_id=$1`, projectID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count after rollback = %d, want 1", count)
	}

	concurrent := domain.Member{ID: "00000000-0000-0000-0000-000000000108", ProjectID: projectID, FullName: "Race"}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m := concurrent
			if i == 1 {
				m.ID = "00000000-0000-0000-0000-000000000109"
			}
			errs <- repo.Register(context.Background(), projectID, []domain.Member{m})
		}(i)
	}
	wg.Wait()
	close(errs)
	success, duplicates := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, application.ErrDuplicateMember) {
			duplicates++
		} else {
			t.Errorf("concurrent Register() error = %v", err)
		}
	}
	if success != 1 || duplicates != 1 {
		t.Fatalf("concurrent results: success=%d duplicate=%d", success, duplicates)
	}
}

func newPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("PROJECTMEMBER_TEST_DATABASE_URL")
	if dsn != "" {
		return testpostgres.OpenIsolated(t, dsn, "PROJECTMEMBER_TEST_DATABASE_URL", "projectmember_test")
	}
	var container *postgres.PostgresContainer
	{
		var err error
		container, err = postgres.Run(ctx, "postgres:18.6-alpine", postgres.WithDatabase("members_test"), postgres.WithUsername("postgres"), postgres.WithPassword("postgres"))
		if err != nil {
			if dockerUnavailable(err) {
				t.Skipf("Docker unavailable: %v", err)
			}
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := container.Terminate(context.Background()); err != nil {
				t.Errorf("terminate container: %v", err)
			}
		})
		dsn, err = container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatal(err)
		}
		return poolForContainer(t, ctx, dsn)
	}
}

func poolForContainer(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if pool.Ping(ctx) == nil {
			return pool
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("PostgreSQL did not become ready")
	return nil
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	root := moduleRoot(t)
	for _, path := range []string{"000001_create_projects.up.sql", "000006_create_project_members.up.sql"} {
		body, err := os.ReadFile(filepath.Join(root, "internal/project/infrastructure/postgres/migrations", path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(body)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
}
func createProject(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := "00000000-0000-0000-0000-000000000001"
	if _, err := pool.Exec(context.Background(), `INSERT INTO projects(id,name,start_date,planned_finish_date) VALUES($1,'Original','2026-01-01','2026-12-31')`, id); err != nil {
		t.Fatal(err)
	}
	return id
}
func stringPointer(s string) *string { return &s }
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
func dockerUnavailable(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "permission denied while trying to connect to the docker api") || strings.Contains(s, "cannot connect to the docker daemon") || strings.Contains(s, "docker is not running")
}
