package http_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/transport/http"
)

type useCase struct {
	err   error
	calls int
}

func (u *useCase) Execute(_ context.Context, c application.RegisterMembersCommand) ([]domain.Member, error) {
	u.calls++
	if u.err != nil {
		return nil, u.err
	}
	m, _ := domain.NewMember("m", c.ProjectID, "Ada", "")
	return []domain.Member{m}, nil
}

func TestRegisterMembersHandlerRejectsNonPostMethods(t *testing.T) {
	uc := &useCase{}
	req := httptest.NewRequest("GET", "/projects/00000000-0000-0000-0000-000000000001/members", nil)
	req.SetPathValue("project_id", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()

	transporthttp.NewRegisterMembersHandler(uc).ServeHTTP(rec, req)

	if rec.Code != 405 {
		t.Fatalf("status = %d, body %s; want 405", rec.Code, rec.Body.String())
	}
	if uc.calls != 0 {
		t.Fatalf("use case called %d times for unsupported method", uc.calls)
	}
}

func TestRegisterMembersHandlerRejectsUnknownJSONFields(t *testing.T) {
	uc := &useCase{}
	req := httptest.NewRequest("POST", "/projects/00000000-0000-0000-0000-000000000001/members", strings.NewReader(`{"members":[{"full_name":"Ada"}],"unexpected":true}`))
	req.SetPathValue("project_id", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()

	transporthttp.NewRegisterMembersHandler(uc).ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("status = %d, body %s; want 400", rec.Code, rec.Body.String())
	}
	if uc.calls != 0 {
		t.Fatalf("use case called %d times for unknown JSON field", uc.calls)
	}
}

func TestRegisterMembersHandlerResponses(t *testing.T) {
	for _, tc := range []struct {
		name, path, body string
		err              error
		status           int
	}{
		{"bad uuid", "/projects/no/members", `{"members":[{"full_name":"Ada"}]}`, nil, 400},
		{"trailing json", "/projects/00000000-0000-0000-0000-000000000001/members", `{"members":[]} {}`, nil, 400},
		{"created", "/projects/00000000-0000-0000-0000-000000000001/members", `{"members":[{"full_name":"Ada"}]}`, nil, 201},
		{"missing", "/projects/00000000-0000-0000-0000-000000000001/members", `{"members":[]}`, application.ErrProjectNotFound, 404},
		{"duplicate", "/projects/00000000-0000-0000-0000-000000000001/members", `{"members":[]}`, application.ErrDuplicateMember, 409},
		{"internal", "/projects/00000000-0000-0000-0000-000000000001/members", `{"members":[]}`, errors.New("secret"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := &useCase{err: tc.err}
			req := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
			req.SetPathValue("project_id", strings.Split(tc.path, "/")[2])
			rec := httptest.NewRecorder()
			transporthttp.NewRegisterMembersHandler(uc).ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
			}
			if tc.status == 400 && uc.calls != 0 {
				t.Fatal("use case called for malformed request")
			}
			if tc.status == 500 && strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("internal details leaked")
			}
		})
	}
}
