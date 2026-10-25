package postgres_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	defectpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/defect/infrastructure/postgres"
)

func TestMapConstraintErrorTranslatesDefectForeignKeys(t *testing.T) {
	cases := map[string]struct {
		code       string
		constraint string
		want       error
	}{
		"story_fk":  {code: "23503", constraint: "defects_story_project_fkey", want: application.ErrStoryNotFound},
		"sprint_fk": {code: "23503", constraint: "defects_sprint_project_fkey", want: application.ErrSprintNotFound},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			pgErr := &pgconn.PgError{Code: tc.code, ConstraintName: tc.constraint}
			if got := defectpostgres.MapConstraintError(pgErr); !errors.Is(got, tc.want) {
				t.Errorf("MapConstraintError(%s/%s) = %v, want %v", tc.code, tc.constraint, got, tc.want)
			}
		})
	}
}

func TestMapConstraintErrorPassesThroughOtherErrors(t *testing.T) {
	cases := map[string]struct {
		code       string
		constraint string
	}{
		"check_violation":               {code: "23514", constraint: "defects_severity_check"},
		"unique_violation":              {code: "23505", constraint: "defects_pkey"},
		"foreign_key_unknown_contraint": {code: "23503", constraint: "other_table_fkey"},
		"foreign_key_no_constraint":     {code: "23503", constraint: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			pgErr := &pgconn.PgError{Code: tc.code, ConstraintName: tc.constraint}
			got := defectpostgres.MapConstraintError(pgErr)
			var mapped *pgconn.PgError
			if !errors.As(got, &mapped) {
				t.Fatalf("MapConstraintError(%v/%v) = %v (%T), want the raw PgError", tc.code, tc.constraint, got, got)
			}
			if mapped.Code != tc.code || mapped.ConstraintName != tc.constraint {
				t.Errorf("raw error mutated: (%s/%s) -> (%s/%s)", tc.code, tc.constraint, mapped.Code, mapped.ConstraintName)
			}
		})
	}
}

func TestMapConstraintErrorPassesThroughNonPgErrors(t *testing.T) {
	wrapped := fmt.Errorf("connection reset by peer")
	if got := defectpostgres.MapConstraintError(wrapped); !errors.Is(got, wrapped) {
		t.Errorf("MapConstraintError(non-pg) = %v, want the original error", got)
	}
}
