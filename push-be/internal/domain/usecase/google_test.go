package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type stubGoogleStore struct {
	integration *domain.GoogleIntegration
	code        *domain.IntegrationCode
	messages    []domain.GoogleMessage
}

func (s *stubGoogleStore) SaveIntegrationCode(ctx context.Context, c *domain.IntegrationCode) error {
	s.code = c
	return nil
}
func (s *stubGoogleStore) GetIntegrationCode(ctx context.Context, code string) (*domain.IntegrationCode, error) {
	if s.code == nil || s.code.Code != code {
		return nil, domain.ErrNotFound
	}
	return s.code, nil
}
func (s *stubGoogleStore) MarkIntegrationCodeUsed(ctx context.Context, code string, at time.Time) error {
	s.code.UsedAt = &at
	return nil
}
func (s *stubGoogleStore) PutGoogleIntegration(ctx context.Context, g *domain.GoogleIntegration) error {
	s.integration = g
	return nil
}
func (s *stubGoogleStore) GetGoogleIntegration(ctx context.Context, userID uuid.UUID) (*domain.GoogleIntegration, error) {
	if s.integration == nil || s.integration.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return s.integration, nil
}
func (s *stubGoogleStore) DeleteGoogleIntegration(ctx context.Context, userID uuid.UUID) error {
	s.integration = nil
	return nil
}
func (s *stubGoogleStore) UpsertGoogleMessage(ctx context.Context, m *domain.GoogleMessage) error {
	s.messages = append(s.messages, *m)
	return nil
}
func (s *stubGoogleStore) GetGoogleMessage(ctx context.Context, userID, id uuid.UUID) (*domain.GoogleMessage, error) {
	for i := range s.messages {
		if s.messages[i].ID == id && s.messages[i].UserID == userID {
			return &s.messages[i], nil
		}
	}
	return nil, domain.ErrNotFound
}
func (s *stubGoogleStore) ListGoogleMessages(ctx context.Context, userID uuid.UUID, applicationID *uuid.UUID, page domain.PageRequest, limit int) ([]domain.GoogleMessage, error) {
	return s.messages, nil
}
func (s *stubGoogleStore) LinkGoogleMessage(ctx context.Context, userID, id, applicationID uuid.UUID) error {
	for i := range s.messages {
		if s.messages[i].ID == id && s.messages[i].UserID == userID {
			s.messages[i].ApplicationID = &applicationID
			return nil
		}
	}
	return domain.ErrNotFound
}
func (s *stubGoogleStore) DeleteGoogleData(ctx context.Context, userID uuid.UUID) error {
	s.messages = nil
	return nil
}

type stubSessionStore struct {
	states map[string]*domain.OAuthState
}

func (s *stubSessionStore) SaveOAuthState(ctx context.Context, st *domain.OAuthState) error {
	if s.states == nil {
		s.states = map[string]*domain.OAuthState{}
	}
	s.states[st.State] = st
	return nil
}
func (s *stubSessionStore) GetOAuthState(ctx context.Context, state string) (*domain.OAuthState, error) {
	if st, ok := s.states[state]; ok {
		return st, nil
	}
	return nil, domain.ErrNotFound
}
func (s *stubSessionStore) DeleteOAuthState(ctx context.Context, state string) error {
	delete(s.states, state)
	return nil
}
func (s *stubSessionStore) SaveExchangeCode(ctx context.Context, c *domain.ExchangeCode) error {
	return nil
}
func (s *stubSessionStore) GetExchangeCode(ctx context.Context, code string) (*domain.ExchangeCode, error) {
	return nil, domain.ErrNotFound
}
func (s *stubSessionStore) MarkExchangeCodeUsed(ctx context.Context, code string, at time.Time) error {
	return nil
}
func (s *stubSessionStore) SaveAccessToken(ctx context.Context, t *domain.AccessToken) error {
	return nil
}
func (s *stubSessionStore) GetAccessToken(ctx context.Context, h string) (*domain.AccessToken, error) {
	return nil, domain.ErrNotFound
}
func (s *stubSessionStore) SaveRefreshToken(ctx context.Context, t *domain.RefreshToken) error {
	return nil
}
func (s *stubSessionStore) GetRefreshToken(ctx context.Context, h string) (*domain.RefreshToken, error) {
	return nil, domain.ErrNotFound
}
func (s *stubSessionStore) RevokeRefreshToken(ctx context.Context, id uuid.UUID, at time.Time) error {
	return nil
}
func (s *stubSessionStore) RevokeAccessTokensForRefresh(ctx context.Context, id uuid.UUID, at time.Time) error {
	return nil
}

type stubTokenCipher struct{}

func (stubTokenCipher) Encrypt(pt string, userID uuid.UUID, purpose string) ([]byte, []byte, error) {
	return []byte("ct:" + pt), []byte("n"), nil
}
func (stubTokenCipher) Decrypt(ct, nonce []byte, userID uuid.UUID, purpose string) (string, error) {
	return string(ct[3:]), nil
}

type stubGoogleClient struct {
	tokens   *domain.GoogleTokens
	messages map[string]*domain.GoogleMessageMeta
	events   []domain.GoogleEvent
}

func (s *stubGoogleClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*domain.GoogleTokens, error) {
	return s.tokens, nil
}
func (s *stubGoogleClient) RefreshAccessToken(ctx context.Context, rt string) (*domain.GoogleTokens, error) {
	return &domain.GoogleTokens{AccessToken: "fresh", ExpiresIn: 3600}, nil
}
func (s *stubGoogleClient) RevokeToken(ctx context.Context, token string) error { return nil }
func (s *stubGoogleClient) ListMessageIDs(ctx context.Context, tok, page string) ([]string, string, string, error) {
	ids := []string{}
	for id := range s.messages {
		ids = append(ids, id)
	}
	return ids, "", "h1", nil
}
func (s *stubGoogleClient) GetMessage(ctx context.Context, tok, id string) (*domain.GoogleMessageMeta, error) {
	return s.messages[id], nil
}
func (s *stubGoogleClient) ListEvents(ctx context.Context, tok, syncToken string) ([]domain.GoogleEvent, string, error) {
	return s.events, "st1", nil
}

func testVerifier() (verifier, challenge string) {
	verifier = "test-verifier-123"
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func newGoogleService(store *stubGoogleStore, sessions *stubSessionStore, gapi domain.GoogleClient) *GoogleService {
	return NewGoogleService(store, sessions, nil, nil, nil, noopUoW{}, stubTokenCipher{}, gapi, GoogleConfig{
		ClientID: "cid", ClientSecret: "csec",
		AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token",
		GmailBeta: true,
	})
}

func TestGoogleConnectValidatesInput(t *testing.T) {
	svc := newGoogleService(&stubGoogleStore{}, &stubSessionStore{}, &stubGoogleClient{})
	_, _, _, err := svc.Connect(context.Background(), uuid.New(), "", "S256", domain.GoogleCallbackURI, "https://cb")
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
	_, _, _, err = svc.Connect(context.Background(), uuid.New(), "ch", "plain", domain.GoogleCallbackURI, "https://cb")
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
	_, _, _, err = svc.Connect(context.Background(), uuid.New(), "ch", "S256", "https://evil.example", "https://cb")
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR for bad redirectUri, got %v", err)
	}
}

func TestGoogleConnectBuildsAuthorizationURL(t *testing.T) {
	sessions := &stubSessionStore{}
	svc := newGoogleService(&stubGoogleStore{}, sessions, &stubGoogleClient{})
	url, state, _, err := svc.Connect(context.Background(), uuid.New(), "ch", "S256", domain.GoogleCallbackURI, "https://host/api/v1/integrations/google/callback")
	if err != nil {
		t.Fatal(err)
	}
	if url == "" || state == "" {
		t.Fatal("empty url or state")
	}
	st := sessions.states[state]
	if st == nil || st.Purpose != domain.OAuthPurposeGoogle {
		t.Fatalf("state not saved with google purpose: %+v", st)
	}
}

func TestGoogleCompletePKCEMismatch(t *testing.T) {
	userID := uuid.New()
	_, challenge := testVerifier()
	payload, _ := json.Marshal(map[string]string{"accessToken": "at"})
	ct, nonce, _ := stubTokenCipher{}.Encrypt(string(payload), userID, "google")
	bundle, _ := json.Marshal(map[string]string{"ciphertext": base64.StdEncoding.EncodeToString(ct), "nonce": base64.StdEncoding.EncodeToString(nonce)})
	store := &stubGoogleStore{code: &domain.IntegrationCode{
		Code: "ic1", UserID: userID, CodeChallenge: challenge,
		Payload: bundle, ExpiresAt: time.Now().Add(time.Minute),
	}}
	svc := newGoogleService(store, &stubSessionStore{}, &stubGoogleClient{})
	_, err := svc.Complete(context.Background(), userID, "ic1", "wrong-verifier")
	if errCode(t, err) != domain.CodeUnauthenticated {
		t.Fatalf("want UNAUTHENTICATED, got %v", err)
	}
}

func TestGoogleCompleteUserMismatch(t *testing.T) {
	verifier, challenge := testVerifier()
	payload, _ := json.Marshal(map[string]string{"accessToken": "at"})
	ct, nonce, _ := stubTokenCipher{}.Encrypt(string(payload), uuid.New(), "google")
	bundle, _ := json.Marshal(map[string]string{"ciphertext": base64.StdEncoding.EncodeToString(ct), "nonce": base64.StdEncoding.EncodeToString(nonce)})
	store := &stubGoogleStore{code: &domain.IntegrationCode{
		Code: "ic1", UserID: uuid.New(), CodeChallenge: challenge,
		Payload: bundle, ExpiresAt: time.Now().Add(time.Minute),
	}}
	svc := newGoogleService(store, &stubSessionStore{}, &stubGoogleClient{})
	_, err := svc.Complete(context.Background(), uuid.New(), "ic1", verifier)
	if errCode(t, err) != domain.CodeUnauthenticated {
		t.Fatalf("want UNAUTHENTICATED, got %v", err)
	}
}

func TestGoogleSyncRequiresConnection(t *testing.T) {
	svc := newGoogleService(&stubGoogleStore{}, &stubSessionStore{}, &stubGoogleClient{})
	_, err := svc.Sync(context.Background(), uuid.New())
	if errCode(t, err) != domain.CodeIntegrationRequired {
		t.Fatalf("want INTEGRATION_REQUIRED, got %v", err)
	}
}
