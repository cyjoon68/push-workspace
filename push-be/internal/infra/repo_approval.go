package infra

import (
	"context"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const approvalCols = `id, user_id, revision, kind, application_id, target_id, target_revision,
	payload_hash, status, expires_at, decided_at, consumed_at, created_at, updated_at`

type ApprovalRepo struct{ d *DB }

func NewApprovalRepo(d *DB) *ApprovalRepo { return &ApprovalRepo{d: d} }

func scanApproval(row interface{ Scan(...any) error }) (*domain.Approval, error) {
	a := &domain.Approval{}
	err := row.Scan(&a.ID, &a.UserID, &a.Revision, &a.Kind, &a.ApplicationID,
		&a.TargetID, &a.TargetRevision, &a.PayloadHash, &a.Status, &a.ExpiresAt,
		&a.DecidedAt, &a.ConsumedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *ApprovalRepo) Create(ctx context.Context, a *domain.Approval) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO approvals (`+approvalCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		a.ID, a.UserID, a.Revision, a.Kind, a.ApplicationID, a.TargetID, a.TargetRevision,
		a.PayloadHash, a.Status, a.ExpiresAt, a.DecidedAt, a.ConsumedAt, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *ApprovalRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Approval, error) {
	a, err := scanApproval(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+approvalCols+` FROM approvals WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return a, nil
}

func (r *ApprovalRepo) List(ctx context.Context, userID uuid.UUID, f domain.ApprovalFilter, limit int) ([]domain.Approval, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if f.ApplicationID != nil {
		c.add("application_id = %s", *f.ApplicationID)
	}
	if f.Status != nil {
		c.add("status = %s", *f.Status)
	}
	c.cursor(f.Page)
	sql, args := c.query(approvalCols, "approvals", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Approval{}
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *ApprovalRepo) Update(ctx context.Context, a *domain.Approval, expectedRevision int64) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE approvals SET revision = revision + 1, status = $3, expires_at = $4,
		 decided_at = $5, consumed_at = $6, updated_at = $7
		 WHERE id = $1 AND user_id = $2 AND revision = $8`,
		a.ID, a.UserID, a.Status, a.ExpiresAt, a.DecidedAt, a.ConsumedAt,
		a.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "approvals", a.ID, a.UserID, expectedRevision)
	}
	return nil
}

func (r *ApprovalRepo) FindActive(ctx context.Context, userID uuid.UUID, kind domain.ApprovalKind, applicationID, targetID uuid.UUID, now time.Time) (*domain.Approval, error) {
	a, err := scanApproval(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+approvalCols+` FROM approvals
		 WHERE user_id = $1 AND kind = $2 AND application_id = $3 AND target_id = $4
		   AND status IN ('APPROVED','PENDING') AND (expires_at IS NULL OR expires_at > $5)
		 ORDER BY created_at DESC LIMIT 1`,
		userID, kind, applicationID, targetID, now))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return a, nil
}
