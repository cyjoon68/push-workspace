package infra

import (
	"context"
	"strconv"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type AiUsageRepo struct{ d *DB }

func NewAiUsageRepo(d *DB) *AiUsageRepo { return &AiUsageRepo{d: d} }

func (r *AiUsageRepo) Create(ctx context.Context, u *domain.AiUsage) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO ai_usage (id, user_id, operation_id, provider, model, managed,
		 input_tokens, output_tokens, cost_micro_credits, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		u.ID, u.UserID, u.OperationID, u.Provider, u.Model, u.Managed,
		u.InputTokens, u.OutputTokens, u.CostMicroCredits, u.Status, u.CreatedAt)
	return err
}

func (r *AiUsageRepo) List(ctx context.Context, userID uuid.UUID, f domain.AiUsageFilter, limit int) ([]domain.AiUsage, error) {
	q := `SELECT id, user_id, operation_id, provider, model, managed,
		input_tokens, output_tokens, cost_micro_credits, status, created_at
		FROM ai_usage WHERE user_id = $1`
	args := []any{userID}
	n := 1
	if f.From != nil {
		n++
		q += ` AND created_at >= $` + itoa(n)
		args = append(args, *f.From)
	}
	if f.To != nil {
		n++
		q += ` AND created_at <= $` + itoa(n)
		args = append(args, *f.To)
	}
	if f.Page.Cursor != nil {
		n++
		q += ` AND (created_at, id) < ($` + itoa(n)
		n++
		q += `, $` + itoa(n) + `)`
		args = append(args, f.Page.Cursor.CreatedAt, f.Page.Cursor.ID)
	}
	q += ` ORDER BY created_at DESC, id DESC`
	n++
	q += ` LIMIT $` + itoa(n)
	args = append(args, limit)
	rows, err := r.d.Q(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AiUsage{}
	for rows.Next() {
		u := domain.AiUsage{}
		if err := rows.Scan(&u.ID, &u.UserID, &u.OperationID, &u.Provider, &u.Model,
			&u.Managed, &u.InputTokens, &u.OutputTokens, &u.CostMicroCredits,
			&u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
