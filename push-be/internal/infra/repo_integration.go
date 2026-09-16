package infra

import (
	"context"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type IntegrationRepo struct{ d *DB }

func NewIntegrationRepo(d *DB) *IntegrationRepo { return &IntegrationRepo{d: d} }

func (r *IntegrationRepo) SaveIntegrationCode(ctx context.Context, c *domain.IntegrationCode) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO integration_codes (code, user_id, code_challenge, payload, expires_at, used_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		c.Code, c.UserID, c.CodeChallenge, c.Payload, c.ExpiresAt, c.UsedAt, c.CreatedAt)
	return err
}

func (r *IntegrationRepo) GetIntegrationCode(ctx context.Context, code string) (*domain.IntegrationCode, error) {
	c := &domain.IntegrationCode{}
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT code, user_id, code_challenge, payload, expires_at, used_at, created_at
		 FROM integration_codes WHERE code = $1`, code).
		Scan(&c.Code, &c.UserID, &c.CodeChallenge, &c.Payload, &c.ExpiresAt, &c.UsedAt, &c.CreatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	return c, nil
}

func (r *IntegrationRepo) MarkIntegrationCodeUsed(ctx context.Context, code string, at time.Time) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE integration_codes SET used_at = $2 WHERE code = $1 AND used_at IS NULL`, code, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *IntegrationRepo) PutGoogleIntegration(ctx context.Context, g *domain.GoogleIntegration) error {
	scopes, err := jv(g.Scopes)
	if err != nil {
		return err
	}
	_, err = r.d.Q(ctx).Exec(ctx,
		`INSERT INTO google_integrations (user_id, access_ciphertext, access_nonce, refresh_ciphertext,
		 refresh_nonce, scopes, token_expires_at, gmail_history_id, calendar_sync_token, last_synced_at,
		 created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 ON CONFLICT (user_id) DO UPDATE SET access_ciphertext=$2, access_nonce=$3,
		 refresh_ciphertext=$4, refresh_nonce=$5, scopes=$6, token_expires_at=$7,
		 gmail_history_id=$8, calendar_sync_token=$9, last_synced_at=$10, updated_at=$12`,
		g.UserID, g.AccessCiphertext, g.AccessNonce, g.RefreshCiphertext, g.RefreshNonce,
		scopes, g.TokenExpiresAt, g.GmailHistoryID, g.CalendarSyncToken, g.LastSyncedAt,
		g.CreatedAt, g.UpdatedAt)
	return err
}

func (r *IntegrationRepo) GetGoogleIntegration(ctx context.Context, userID uuid.UUID) (*domain.GoogleIntegration, error) {
	g := &domain.GoogleIntegration{}
	var scopes []byte
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT user_id, access_ciphertext, access_nonce, refresh_ciphertext, refresh_nonce, scopes,
		 token_expires_at, gmail_history_id, calendar_sync_token, last_synced_at, created_at, updated_at
		 FROM google_integrations WHERE user_id = $1`, userID).
		Scan(&g.UserID, &g.AccessCiphertext, &g.AccessNonce, &g.RefreshCiphertext, &g.RefreshNonce,
			&scopes, &g.TokenExpiresAt, &g.GmailHistoryID, &g.CalendarSyncToken, &g.LastSyncedAt,
			&g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	g.Scopes = []string{}
	if err := unj(scopes, &g.Scopes); err != nil {
		return nil, err
	}
	return g, nil
}

func (r *IntegrationRepo) DeleteGoogleIntegration(ctx context.Context, userID uuid.UUID) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`DELETE FROM google_integrations WHERE user_id = $1`, userID)
	return err
}

const googleMessageCols = `id, user_id, external_id, thread_id, application_id, sender, subject, snippet, received_at, created_at`

func scanGoogleMessage(row interface{ Scan(...any) error }) (*domain.GoogleMessage, error) {
	m := &domain.GoogleMessage{}
	err := row.Scan(&m.ID, &m.UserID, &m.ExternalID, &m.ThreadID, &m.ApplicationID,
		&m.Sender, &m.Subject, &m.Snippet, &m.ReceivedAt, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *IntegrationRepo) UpsertGoogleMessage(ctx context.Context, m *domain.GoogleMessage) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO google_messages (`+googleMessageCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 ON CONFLICT (user_id, external_id) DO UPDATE
		 SET thread_id=$4, sender=$6, subject=$7, snippet=$8, received_at=$9`,
		m.ID, m.UserID, m.ExternalID, m.ThreadID, m.ApplicationID,
		m.Sender, m.Subject, m.Snippet, m.ReceivedAt, m.CreatedAt)
	return err
}

func (r *IntegrationRepo) GetGoogleMessage(ctx context.Context, userID, id uuid.UUID) (*domain.GoogleMessage, error) {
	m, err := scanGoogleMessage(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+googleMessageCols+` FROM google_messages WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return m, nil
}

func (r *IntegrationRepo) ListGoogleMessages(ctx context.Context, userID uuid.UUID, applicationID *uuid.UUID, page domain.PageRequest, limit int) ([]domain.GoogleMessage, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if applicationID != nil {
		c.add("application_id = %s", *applicationID)
	}
	c.cursor(page)
	sql, args := c.query(googleMessageCols, "google_messages", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GoogleMessage{}
	for rows.Next() {
		m, err := scanGoogleMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *IntegrationRepo) LinkGoogleMessage(ctx context.Context, userID, id, applicationID uuid.UUID) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE google_messages SET application_id = $3 WHERE id = $1 AND user_id = $2`,
		id, userID, applicationID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *IntegrationRepo) DeleteGoogleData(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.d.Q(ctx).Exec(ctx,
		`DELETE FROM google_messages WHERE user_id = $1`, userID); err != nil {
		return err
	}
	_, err := r.d.Q(ctx).Exec(ctx,
		`DELETE FROM calendar_events WHERE user_id = $1 AND source = 'GOOGLE'`, userID)
	return err
}
