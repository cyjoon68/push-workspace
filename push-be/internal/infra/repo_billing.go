package infra

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"push-be/internal/domain"
)

type BillingRepo struct{ d *DB }

func NewBillingRepo(d *DB) *BillingRepo { return &BillingRepo{d: d} }

func (r *BillingRepo) RecordStripeEvent(ctx context.Context, eventID, typ string, at time.Time) (bool, error) {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO stripe_events (event_id, type, processed_at) VALUES ($1,$2,$3)
		 ON CONFLICT (event_id) DO NOTHING`, eventID, typ, at)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *BillingRepo) UpdateSubscription(ctx context.Context, userID uuid.UUID, plan, status string, customerID *string, periodEndsAt *time.Time) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE users SET plan = $2, subscription_status = $3, stripe_customer_id = COALESCE($4, stripe_customer_id),
		 period_ends_at = $5 WHERE id = $1`,
		userID, plan, status, customerID, periodEndsAt)
	return err
}

func (r *BillingRepo) UpdateSubscriptionByCustomer(ctx context.Context, customerID, status string, periodEndsAt *time.Time) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE users SET subscription_status = $2, period_ends_at = $3 WHERE stripe_customer_id = $1`,
		customerID, status, periodEndsAt)
	return err
}

func (r *BillingRepo) UserIDByStripeCustomer(ctx context.Context, customerID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT id FROM users WHERE stripe_customer_id = $1`, customerID).Scan(&id)
	if err != nil {
		return uuid.Nil, mapGetErr(err)
	}
	return id, nil
}

func (r *BillingRepo) AppendLedger(ctx context.Context, e *domain.LedgerEntry) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO billing_ledger (id, user_id, type, amount_micro_credits, balance_after, reference_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.UserID, e.Type, e.AmountMicroCredits, e.BalanceAfter, e.ReferenceID, e.CreatedAt)
	return err
}

const ledgerCols = `id, user_id, type, amount_micro_credits, balance_after, reference_id, created_at`

func (r *BillingRepo) ListLedger(ctx context.Context, userID uuid.UUID, page domain.PageRequest, limit int) ([]domain.LedgerEntry, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.cursor(page)
	sql, args := c.query(ledgerCols, "billing_ledger", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.LedgerEntry{}
	for rows.Next() {
		e := domain.LedgerEntry{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.Type, &e.AmountMicroCredits, &e.BalanceAfter, &e.ReferenceID, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *BillingRepo) LastBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	var bal int64
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT balance_after FROM billing_ledger WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC LIMIT 1`, userID).Scan(&bal)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return bal, nil
}
