package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

type OAuthClient interface {
	ExchangeCode(ctx context.Context, cfg ProviderConfig, code, redirectURI string) (string, error)
	FetchIdentity(ctx context.Context, cfg ProviderConfig, accessToken string) (subject, displayName string, err error)
}

type AuthService struct {
	users       domain.UserStore
	sessions    domain.SessionStore
	uow         domain.UnitOfWork
	oauth       OAuthClient
	providers   map[string]ProviderConfig
	appEnv      string
	devToken    string
	devUserID   uuid.UUID
	allowedURIs map[string]bool
}

func NewAuthService(users domain.UserStore, sessions domain.SessionStore, uow domain.UnitOfWork, oauth OAuthClient, providers map[string]ProviderConfig, appEnv, devToken string, devUserID uuid.UUID) *AuthService {
	return &AuthService{
		users: users, sessions: sessions, uow: uow, oauth: oauth,
		providers: providers, appEnv: appEnv, devToken: devToken, devUserID: devUserID,
		allowedURIs: map[string]bool{domain.AuthCallbackURI: true},
	}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func pkceS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *AuthService) StartOAuth(ctx context.Context, provider, codeChallenge, method, redirectURI, providerCallbackURL string) (string, string, time.Time, error) {
	if !domain.ValidOAuthProvider(provider) {
		return "", "", time.Time{}, domain.NotFound()
	}
	cfg, ok := s.providers[provider]
	if !ok || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return "", "", time.Time{}, domain.NotConfigured("oauth provider " + provider + " is not configured")
	}
	if method != "S256" {
		return "", "", time.Time{}, domain.ValidationField("codeChallengeMethod", "must be S256")
	}
	if codeChallenge == "" {
		return "", "", time.Time{}, domain.ValidationField("codeChallenge", "required")
	}
	if !s.allowedURIs[redirectURI] {
		return "", "", time.Time{}, domain.ValidationField("redirectUri", "not allowed")
	}
	state, err := randomToken()
	if err != nil {
		return "", "", time.Time{}, domain.Internal()
	}
	now := time.Now().UTC()
	rec := &domain.OAuthState{
		State: state, Provider: provider, CodeChallenge: codeChallenge,
		RedirectURI: providerCallbackURL, ExpiresAt: now.Add(domain.OAuthStateTTL), CreatedAt: now,
	}
	if err := s.sessions.SaveOAuthState(ctx, rec); err != nil {
		return "", "", time.Time{}, domain.Internal()
	}
	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", providerCallbackURL)
	q.Set("response_type", "code")
	q.Set("state", state)
	if provider == "google" {
		q.Set("scope", "openid email profile")
		q.Set("access_type", "offline")
	} else {
		q.Set("scope", "read:user user:email")
	}
	return cfg.AuthURL + "?" + q.Encode(), state, rec.ExpiresAt, nil
}

func (s *AuthService) HandleCallback(ctx context.Context, provider, code, state string) (string, error) {
	if !domain.ValidOAuthProvider(provider) {
		return "", domain.NotFound()
	}
	cfg, ok := s.providers[provider]
	if !ok || cfg.ClientID == "" {
		return "", domain.NotConfigured("oauth provider " + provider + " is not configured")
	}
	rec, err := s.sessions.GetOAuthState(ctx, state)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", domain.Unauthenticated("invalid state")
		}
		return "", domain.Internal()
	}
	now := time.Now().UTC()
	_ = s.sessions.DeleteOAuthState(ctx, state)
	if now.After(rec.ExpiresAt) || rec.Provider != provider {
		return "", domain.Unauthenticated("state expired")
	}
	accessToken, err := s.oauth.ExchangeCode(ctx, cfg, code, rec.RedirectURI)
	if err != nil {
		return "", domain.ProviderError("provider token exchange failed")
	}
	subject, displayName, err := s.oauth.FetchIdentity(ctx, cfg, accessToken)
	if err != nil {
		return "", domain.ProviderError("provider identity fetch failed")
	}
	var exchangeCode string
	err = s.uow.Do(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByProvider(ctx, provider, subject)
		if err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
				return err
			}
			user = &domain.User{
				ID: uuid.New(), Provider: provider, ProviderSubject: subject,
				DisplayName: displayName, Locale: "ko", CreatedAt: now,
			}
			if err := s.users.Create(ctx, user); err != nil {
				return err
			}
		}
		exchangeCode, err = randomToken()
		if err != nil {
			return err
		}
		return s.sessions.SaveExchangeCode(ctx, &domain.ExchangeCode{
			Code: exchangeCode, UserID: user.ID, CodeChallenge: rec.CodeChallenge,
			ExpiresAt: now.Add(domain.ExchangeCodeTTL), CreatedAt: now,
		})
	})
	if err != nil {
		return "", domain.Internal()
	}
	return domain.AuthCallbackURI + "?code=" + url.QueryEscape(exchangeCode), nil
}

func (s *AuthService) issueSession(ctx context.Context, userID uuid.UUID, now time.Time) (*domain.Session, error) {
	access, err := randomToken()
	if err != nil {
		return nil, err
	}
	refresh, err := randomToken()
	if err != nil {
		return nil, err
	}
	rt := &domain.RefreshToken{
		ID: uuid.New(), UserID: userID, TokenHash: tokenHash(refresh),
		ExpiresAt: now.Add(domain.RefreshTokenTTL), CreatedAt: now,
	}
	if err := s.sessions.SaveRefreshToken(ctx, rt); err != nil {
		return nil, err
	}
	at := &domain.AccessToken{
		ID: uuid.New(), UserID: userID, TokenHash: tokenHash(access),
		RefreshTokenID: rt.ID, ExpiresAt: now.Add(time.Duration(domain.AccessTokenTTLSeconds) * time.Second), CreatedAt: now,
	}
	if err := s.sessions.SaveAccessToken(ctx, at); err != nil {
		return nil, err
	}
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &domain.Session{
		AccessToken: access, RefreshToken: refresh,
		ExpiresIn: domain.AccessTokenTTLSeconds, User: user,
	}, nil
}

func (s *AuthService) Exchange(ctx context.Context, code, codeVerifier string) (*domain.Session, error) {
	rec, err := s.sessions.GetExchangeCode(ctx, code)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.Unauthenticated("invalid code")
		}
		return nil, domain.Internal()
	}
	now := time.Now().UTC()
	if rec.UsedAt != nil || now.After(rec.ExpiresAt) {
		return nil, domain.Unauthenticated("code expired or already used")
	}
	if codeVerifier == "" || pkceS256(codeVerifier) != rec.CodeChallenge {
		return nil, domain.Unauthenticated("pkce verification failed")
	}
	var session *domain.Session
	err = s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.sessions.MarkExchangeCodeUsed(ctx, code, now); err != nil {
			return err
		}
		sess, err := s.issueSession(ctx, rec.UserID, now)
		if err != nil {
			return err
		}
		session = sess
		return nil
	})
	if err != nil {
		return nil, domain.Internal()
	}
	return session, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*domain.Session, error) {
	rec, err := s.sessions.GetRefreshToken(ctx, tokenHash(refreshToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.Unauthenticated("invalid refresh token")
		}
		return nil, domain.Internal()
	}
	now := time.Now().UTC()
	if rec.RevokedAt != nil {
		return nil, domain.Unauthenticated("refresh token revoked")
	}
	if now.After(rec.ExpiresAt) {
		return nil, domain.TokenExpired()
	}
	var session *domain.Session
	err = s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.sessions.RevokeRefreshToken(ctx, rec.ID, now); err != nil {
			return err
		}
		if err := s.sessions.RevokeAccessTokensForRefresh(ctx, rec.ID, now); err != nil {
			return err
		}
		sess, err := s.issueSession(ctx, rec.UserID, now)
		if err != nil {
			return err
		}
		session = sess
		return nil
	})
	if err != nil {
		return nil, domain.Internal()
	}
	return session, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	rec, err := s.sessions.GetRefreshToken(ctx, tokenHash(refreshToken))
	if err != nil {
		return nil
	}
	now := time.Now().UTC()
	_ = s.sessions.RevokeRefreshToken(ctx, rec.ID, now)
	_ = s.sessions.RevokeAccessTokensForRefresh(ctx, rec.ID, now)
	return nil
}

func (s *AuthService) ResolveAccessToken(ctx context.Context, token string) (*domain.User, error) {
	if s.appEnv == "development" && s.devToken != "" && token == s.devToken {
		user, err := s.users.Get(ctx, s.devUserID)
		if err != nil {
			return nil, domain.Unauthenticated("dev user not seeded")
		}
		return user, nil
	}
	rec, err := s.sessions.GetAccessToken(ctx, tokenHash(token))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.Unauthenticated("invalid access token")
		}
		return nil, domain.Internal()
	}
	if time.Now().UTC().After(rec.ExpiresAt) {
		return nil, domain.TokenExpired()
	}
	user, err := s.users.Get(ctx, rec.UserID)
	if err != nil {
		return nil, domain.Unauthenticated("invalid access token")
	}
	return user, nil
}

func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return nil, domain.NotFound()
	}
	return user, nil
}
