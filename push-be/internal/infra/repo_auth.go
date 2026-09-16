package infra

import (
	"context"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type UserRepo struct{ d *DB }

func NewUserRepo(d *DB) *UserRepo { return &UserRepo{d: d} }

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO users (id, provider, provider_subject, display_name, locale, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		u.ID, u.Provider, u.ProviderSubject, u.DisplayName, u.Locale, u.CreatedAt)
	return err
}

const userCols = `id, provider, provider_subject, display_name, locale, plan,
	subscription_status, stripe_customer_id, period_ends_at, created_at`

func scanUser(row interface{ Scan(...any) error }) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Provider, &u.ProviderSubject, &u.DisplayName, &u.Locale,
		&u.Plan, &u.SubscriptionStatus, &u.StripeCustomerID, &u.PeriodEndsAt, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) Get(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := scanUser(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id = $1`, id))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return u, nil
}

func (r *UserRepo) GetByProvider(ctx context.Context, provider, subject string) (*domain.User, error) {
	u, err := scanUser(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE provider = $1 AND provider_subject = $2`, provider, subject))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return u, nil
}

func (r *UserRepo) UpsertDevUser(ctx context.Context, id uuid.UUID) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO users (id, provider, provider_subject, display_name, locale, created_at)
		 VALUES ($1,'dev','dev','Dev User','ko',now())
		 ON CONFLICT (id) DO NOTHING`, id)
	return err
}

type SessionRepo struct{ d *DB }

func NewSessionRepo(d *DB) *SessionRepo { return &SessionRepo{d: d} }

func (r *SessionRepo) SaveOAuthState(ctx context.Context, s *domain.OAuthState) error {
	purpose := s.Purpose
	if purpose == "" {
		purpose = domain.OAuthPurposeLogin
	}
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO oauth_states (state, provider, code_challenge, redirect_uri, expires_at, created_at, user_id, purpose)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		s.State, s.Provider, s.CodeChallenge, s.RedirectURI, s.ExpiresAt, s.CreatedAt, s.UserID, purpose)
	return err
}

func (r *SessionRepo) GetOAuthState(ctx context.Context, state string) (*domain.OAuthState, error) {
	s := &domain.OAuthState{}
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT state, provider, code_challenge, redirect_uri, expires_at, created_at, user_id, purpose
		 FROM oauth_states WHERE state = $1`, state).
		Scan(&s.State, &s.Provider, &s.CodeChallenge, &s.RedirectURI, &s.ExpiresAt, &s.CreatedAt, &s.UserID, &s.Purpose)
	if err != nil {
		return nil, mapGetErr(err)
	}
	return s, nil
}

func (r *SessionRepo) DeleteOAuthState(ctx context.Context, state string) error {
	_, err := r.d.Q(ctx).Exec(ctx, `DELETE FROM oauth_states WHERE state = $1`, state)
	return err
}

func (r *SessionRepo) SaveExchangeCode(ctx context.Context, c *domain.ExchangeCode) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO exchange_codes (code, user_id, code_challenge, expires_at, used_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		c.Code, c.UserID, c.CodeChallenge, c.ExpiresAt, c.UsedAt, c.CreatedAt)
	return err
}

func (r *SessionRepo) GetExchangeCode(ctx context.Context, code string) (*domain.ExchangeCode, error) {
	c := &domain.ExchangeCode{}
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT code, user_id, code_challenge, expires_at, used_at, created_at
		 FROM exchange_codes WHERE code = $1`, code).
		Scan(&c.Code, &c.UserID, &c.CodeChallenge, &c.ExpiresAt, &c.UsedAt, &c.CreatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	return c, nil
}

func (r *SessionRepo) MarkExchangeCodeUsed(ctx context.Context, code string, at time.Time) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE exchange_codes SET used_at = $2 WHERE code = $1 AND used_at IS NULL`, code, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepo) SaveAccessToken(ctx context.Context, t *domain.AccessToken) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO access_tokens (id, user_id, token_hash, refresh_token_id, expires_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		t.ID, t.UserID, t.TokenHash, t.RefreshTokenID, t.ExpiresAt, t.CreatedAt)
	return err
}

func (r *SessionRepo) GetAccessToken(ctx context.Context, tokenHash string) (*domain.AccessToken, error) {
	t := &domain.AccessToken{}
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT id, user_id, token_hash, refresh_token_id, expires_at, created_at
		 FROM access_tokens WHERE token_hash = $1`, tokenHash).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.RefreshTokenID, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	return t, nil
}

func (r *SessionRepo) SaveRefreshToken(ctx context.Context, t *domain.RefreshToken) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		t.ID, t.UserID, t.TokenHash, t.ExpiresAt, t.RevokedAt, t.CreatedAt)
	return err
}

func (r *SessionRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	t := &domain.RefreshToken{}
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, tokenHash).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	return t, nil
}

func (r *SessionRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`, id, at)
	return err
}

func (r *SessionRepo) RevokeAccessTokensForRefresh(ctx context.Context, refreshTokenID uuid.UUID, at time.Time) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE access_tokens SET expires_at = $2 WHERE refresh_token_id = $1`, refreshTokenID, at)
	return err
}
