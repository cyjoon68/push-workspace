package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type AiKeyRepo struct{ d *DB }

func NewAiKeyRepo(d *DB) *AiKeyRepo { return &AiKeyRepo{d: d} }

func (r *AiKeyRepo) Put(ctx context.Context, k *domain.AiKey) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO ai_keys (user_id, provider, last_four, ciphertext, nonce, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (user_id, provider) DO UPDATE
		 SET last_four = $3, ciphertext = $4, nonce = $5, updated_at = $6`,
		k.UserID, k.Provider, k.LastFour, k.Ciphertext, k.Nonce, k.UpdatedAt)
	return err
}

func (r *AiKeyRepo) Get(ctx context.Context, userID uuid.UUID, provider string) (*domain.AiKey, error) {
	k := &domain.AiKey{}
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT user_id, provider, last_four, ciphertext, nonce, updated_at
		 FROM ai_keys WHERE user_id = $1 AND provider = $2`, userID, provider).
		Scan(&k.UserID, &k.Provider, &k.LastFour, &k.Ciphertext, &k.Nonce, &k.UpdatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	return k, nil
}

func (r *AiKeyRepo) List(ctx context.Context, userID uuid.UUID) ([]domain.AiKey, error) {
	rows, err := r.d.Q(ctx).Query(ctx,
		`SELECT user_id, provider, last_four, ciphertext, nonce, updated_at
		 FROM ai_keys WHERE user_id = $1 ORDER BY provider`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AiKey{}
	for rows.Next() {
		k := domain.AiKey{}
		if err := rows.Scan(&k.UserID, &k.Provider, &k.LastFour, &k.Ciphertext, &k.Nonce, &k.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *AiKeyRepo) Delete(ctx context.Context, userID uuid.UUID, provider string) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`DELETE FROM ai_keys WHERE user_id = $1 AND provider = $2`, userID, provider)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AiKeyRepo) Has(ctx context.Context, userID uuid.UUID, provider string) bool {
	var n int
	if err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM ai_keys WHERE user_id = $1 AND provider = $2`, userID, provider).Scan(&n); err != nil {
		return false
	}
	return n > 0
}
