package postgres_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	projectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/project/infrastructure/postgres"
	memberpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/infrastructure/postgres"
)

func TestMembersHTTPWithMigratedPostgres(t *testing.T) {
	pool := newPostgresPool(t)
	applyMigrations(t, pool)
	projectID := createProject(t, pool)
	if _, err := pool.Exec(context.Background(), `CREATE TABLE schema_migrations (version bigint NOT NULL, dirty boolean NOT NULL); INSERT INTO schema_migrations(version, dirty) VALUES (6, false)`); err != nil {
		t.Fatal(err)
	}
	handler := api.NewHTTPHandlerWithDependencies(
		projectpostgres.NewPostgresProjectRepository(pool), api.NewProjectID,
		api.HTTPDependencies{Members: &api.MemberDependencies{Repository: memberpostgres.NewPostgresMemberRepository(pool), GenerateID: api.NewProjectID}},
	)

	body := `{"members":[{"full_name":"Alex Example","email":"alex@example.com"},{"full_name":"Sam Example"}]}`
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/members", strings.NewReader(body)))
	var result struct {
		Members []struct {
			ID        string  `json:"id"`
			ProjectID string  `json:"project_id"`
			FullName  string  `json:"full_name"`
			Email     *string `json:"email"`
		} `json:"members"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode success response: %v; body=%s", err, created.Body.String())
	}
	if created.Code != http.StatusCreated || len(result.Members) != 2 {
		t.Fatalf("create status=%d members=%+v body=%s", created.Code, result.Members, created.Body.String())
	}
	if result.Members[0].ID == "" || result.Members[0].ProjectID != projectID || result.Members[0].FullName != "Alex Example" || result.Members[0].Email == nil || *result.Members[0].Email != "alex@example.com" {
		t.Fatalf("first response member = %+v", result.Members[0])
	}
	if result.Members[1].ID == "" || result.Members[1].ProjectID != projectID || result.Members[1].FullName != "Sam Example" {
		t.Fatalf("second response member = %+v", result.Members[1])
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM project_members WHERE project_id=$1`, projectID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("persisted count=%d error=%v, want 2", count, err)
	}

	duplicate := httptest.NewRecorder()
	handler.ServeHTTP(duplicate, httptest.NewRequest(http.MethodPost, "/projects/"+projectID+"/members", strings.NewReader(`{"members":[{"full_name":"Alex Example","email":"alex@example.com"}]}`)))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s, want 409", duplicate.Code, duplicate.Body.String())
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM project_members WHERE project_id=$1`, projectID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("count after duplicate=%d error=%v, want 2", count, err)
	}

	missingID := "00000000-0000-0000-0000-000000000099"
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/projects/"+missingID+"/members", strings.NewReader(`{"members":[{"full_name":"Missing Project Member"}]}`)))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing project status=%d body=%s, want 404", missing.Code, missing.Body.String())
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM project_members`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("count after missing-project request=%d error=%v, want 2", count, err)
	}
}
