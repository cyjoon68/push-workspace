package infra

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"push-be/internal/domain"
)

type IdempotencyRepo struct{ d *DB }

func NewIdempotencyRepo(d *DB) *IdempotencyRepo { return &IdempotencyRepo{d: d} }

func (r *IdempotencyRepo) InsertPending(ctx context.Context, rec *domain.IdempotencyRecord) (bool, error) {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO idempotency_keys (id, user_id, method, path, key, request_hash, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		rec.ID, rec.UserID, rec.Method, rec.Path, rec.Key, rec.RequestHash, rec.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isUniqueViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}

func (r *IdempotencyRepo) Get(ctx context.Context, userID uuid.UUID, method, path string, key uuid.UUID) (*domain.IdempotencyRecord, error) {
	rec := &domain.IdempotencyRecord{}
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT id, user_id, key, method, path, request_hash, response_status, response_body, created_at
		 FROM idempotency_keys WHERE user_id = $1 AND method = $2 AND path = $3 AND key = $4`,
		userID, method, path, key).
		Scan(&rec.ID, &rec.UserID, &rec.Key, &rec.Method, &rec.Path, &rec.RequestHash,
			&rec.ResponseStatus, &rec.ResponseBody, &rec.CreatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	return rec, nil
}

func (r *IdempotencyRepo) Complete(ctx context.Context, id uuid.UUID, status int, body []byte) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE idempotency_keys SET response_status = $2, response_body = $3 WHERE id = $1`,
		id, status, body)
	return err
}

func (r *IdempotencyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.d.Q(ctx).Exec(ctx, `DELETE FROM idempotency_keys WHERE id = $1`, id)
	return err
}
