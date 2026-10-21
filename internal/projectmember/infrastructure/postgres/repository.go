package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
)

// PostgresMemberRepository stores project members in PostgreSQL.
type PostgresMemberRepository struct{ pool *pgxpool.Pool }

// NewPostgresMemberRepository creates a PostgreSQL-backed member repository.
func NewPostgresMemberRepository(pool *pgxpool.Pool) *PostgresMemberRepository {
	return &PostgresMemberRepository{pool: pool}
}

// Register inserts a complete member batch atomically.
func (r *PostgresMemberRepository) Register(ctx context.Context, projectID string, members []domain.Member) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)`, projectID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return application.ErrProjectNotFound
	}

	for _, member := range members {
		var email any
		if member.Email != nil {
			email = *member.Email
		}
		_, err := tx.Exec(ctx, `INSERT INTO project_members (id, project_id, full_name, email) VALUES ($1, $2, $3, $4)`, member.ID, projectID, member.FullName, email)
		if err != nil {
			if isMemberUniqueViolation(err) {
				return application.ErrDuplicateMember
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListByProject returns the members of an existing project ordered by name and identifier.
// A project without members is not an error; the returned slice is never nil.
func (r *PostgresMemberRepository) ListByProject(ctx context.Context, projectID string) ([]domain.Member, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)`, projectID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, application.ErrProjectNotFound
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id::text, project_id::text, full_name, email
		FROM project_members
		WHERE project_id = $1
		ORDER BY full_name ASC, id ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]domain.Member, 0)
	for rows.Next() {
		var member domain.Member
		var email *string
		if err := rows.Scan(&member.ID, &member.ProjectID, &member.FullName, &email); err != nil {
			return nil, err
		}
		member.Email = email
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return members, nil
}

func isMemberUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	switch pgErr.ConstraintName {
	case "project_members_identity_with_email", "project_members_identity_without_email", "project_members_pkey":
		return true
	default:
		return false
	}
}
