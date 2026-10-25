package postgres_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/application"
	effortpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/effort/infrastructure/postgres"
)

// TestMapConstraintError cubre la traducción de violaciones de clave foránea de effort_records
// con errores de PostgreSQL sintéticos: la carrera real (una eliminación concurrente entre la
// verificación y el INSERT) no se puede reproducir de forma determinista.
func TestMapConstraintError(t *testing.T) {
	genericError := errors.New("connection reset by peer")
	foreignKey := func(constraint string) *pgconn.PgError {
		return &pgconn.PgError{Code: "23503", ConstraintName: constraint}
	}
	unknownForeignKey := foreignKey("effort_records_other_fkey")
	checkViolation := &pgconn.PgError{Code: "23514", ConstraintName: "effort_records_hours_range"}

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "story foreign key maps to story not found", err: foreignKey("effort_records_story_project_fkey"), want: application.ErrStoryNotFound},
		{name: "member foreign key maps to member not found", err: foreignKey("effort_records_member_project_fkey"), want: application.ErrMemberNotFound},
		{name: "task foreign key maps to task not found", err: foreignKey("effort_records_task_story_project_fkey"), want: application.ErrTaskNotFound},
		{name: "wrapped foreign key is recognized through errors.As", err: fmt.Errorf("insert effort record: %w", foreignKey("effort_records_member_project_fkey")), want: application.ErrMemberNotFound},
		{name: "foreign key with another constraint name is returned unchanged", err: unknownForeignKey, want: unknownForeignKey},
		{name: "check violation is not mapped", err: checkViolation, want: checkViolation},
		{name: "non PostgreSQL error is returned unchanged", err: genericError, want: genericError},
		{name: "nil stays nil", err: nil, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := effortpostgres.MapConstraintError(test.err)

			if test.want == nil {
				if got != nil {
					t.Fatalf("MapConstraintError(%v) = %v, want nil", test.err, got)
				}
				return
			}
			if !errors.Is(got, test.want) {
				t.Errorf("MapConstraintError(%v) = %v, want %v", test.err, got, test.want)
			}
			if got != test.want {
				t.Errorf("MapConstraintError(%v) = %#v, want identical error %#v", test.err, got, test.want)
			}
		})
	}
}
