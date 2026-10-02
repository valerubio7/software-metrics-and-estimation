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
