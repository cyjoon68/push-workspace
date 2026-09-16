package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const googleCipherPurpose = "google"

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	GmailBeta    bool
}

func (c GoogleConfig) Configured() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

type GoogleService struct {
	store        domain.GoogleStore
	sessions     domain.SessionStore
	calendar     domain.CalendarEventStore
	applications domain.ApplicationStore
	ops          domain.OperationStore
	uow          domain.UnitOfWork
	cipher       domain.TokenCipher
	gapi         domain.GoogleClient
	cfg          GoogleConfig
}

func NewGoogleService(store domain.GoogleStore, sessions domain.SessionStore, calendar domain.CalendarEventStore, applications domain.ApplicationStore, ops domain.OperationStore, uow domain.UnitOfWork, cipher domain.TokenCipher, gapi domain.GoogleClient, cfg GoogleConfig) *GoogleService {
	return &GoogleService{
		store: store, sessions: sessions, calendar: calendar, applications: applications,
		ops: ops, uow: uow, cipher: cipher, gapi: gapi, cfg: cfg,
	}
}

func (s *GoogleService) Status(ctx context.Context, userID uuid.UUID) (*domain.GoogleStatus, error) {
	st := &domain.GoogleStatus{
		Enabled: s.cfg.Configured(), Connected: false,
		Scopes: []string{}, CalendarStatus: "DISCONNECTED",
	}
	if !s.cfg.GmailBeta {
		st.GmailStatus = "DISABLED"
	} else {
		st.GmailStatus = "DISCONNECTED"
	}
	gi, err := s.store.GetGoogleIntegration(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return st, nil
		}
		return nil, domain.Internal()
	}
	st.Connected = true
	st.Scopes = gi.Scopes
	st.LastSyncedAt = gi.LastSyncedAt
	if s.cfg.GmailBeta && gi.HasScope(domain.GoogleScopeGmail) {
		st.GmailStatus = "CONNECTED"
	}
	if gi.HasScope(domain.GoogleScopeCalendar) {
		st.CalendarStatus = "CONNECTED"
	}
	return st, nil
}

func (s *GoogleService) Connect(ctx context.Context, userID uuid.UUID, codeChallenge, method, redirectURI, callbackURL string) (string, string, time.Time, error) {
	if !s.cfg.Configured() {
		return "", "", time.Time{}, domain.NotConfigured("google integration is not configured")
	}
	if s.cipher == nil {
		return "", "", time.Time{}, domain.NotConfigured("token encryption is not configured")
	}
	if method != "S256" {
		return "", "", time.Time{}, domain.ValidationField("codeChallengeMethod", "must be S256")
	}
	if codeChallenge == "" {
		return "", "", time.Time{}, domain.ValidationField("codeChallenge", "required")
	}
	if redirectURI != domain.GoogleCallbackURI {
		return "", "", time.Time{}, domain.ValidationField("redirectUri", "not allowed")
	}
	state, err := randomToken()
	if err != nil {
		return "", "", time.Time{}, domain.Internal()
	}
	now := time.Now().UTC()
	rec := &domain.OAuthState{
		State: state, Provider: "google", Purpose: domain.OAuthPurposeGoogle,
		UserID: &userID, CodeChallenge: codeChallenge, RedirectURI: callbackURL,
		ExpiresAt: now.Add(domain.OAuthStateTTL), CreatedAt: now,
	}
	if err := s.sessions.SaveOAuthState(ctx, rec); err != nil {
		return "", "", time.Time{}, domain.Internal()
	}
	q := url.Values{}
	q.Set("client_id", s.cfg.ClientID)
	q.Set("redirect_uri", callbackURL)
	q.Set("response_type", "code")
	q.Set("state", state)
	q.Set("scope", domain.GoogleScopeGmail+" "+domain.GoogleScopeCalendar)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	return s.cfg.AuthURL + "?" + q.Encode(), state, rec.ExpiresAt, nil
}

type googleTokenPayload struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expiresIn"`
}

func (s *GoogleService) HandleCallback(ctx context.Context, code, state string) (string, error) {
	if !s.cfg.Configured() {
		return "", domain.NotConfigured("google integration is not configured")
	}
	if s.cipher == nil {
		return "", domain.NotConfigured("token encryption is not configured")
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
	if now.After(rec.ExpiresAt) || rec.Purpose != domain.OAuthPurposeGoogle || rec.UserID == nil {
		return "", domain.Unauthenticated("state expired")
	}
	tokens, err := s.gapi.ExchangeCode(ctx, code, rec.RedirectURI)
	if err != nil {
		return "", domain.ProviderError("provider token exchange failed")
	}
	payload, _ := json.Marshal(googleTokenPayload{
		AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken,
		Scope: tokens.Scope, ExpiresIn: tokens.ExpiresIn,
	})
	ct, nonce, err := s.cipher.Encrypt(string(payload), *rec.UserID, googleCipherPurpose)
	if err != nil {
		return "", domain.Internal()
	}
	bundle, _ := json.Marshal(map[string]string{
		"ciphertext": base64.StdEncoding.EncodeToString(ct),
		"nonce":      base64.StdEncoding.EncodeToString(nonce),
	})
	integrationCode, err := randomToken()
	if err != nil {
		return "", domain.Internal()
	}
	if err := s.store.SaveIntegrationCode(ctx, &domain.IntegrationCode{
		Code: integrationCode, UserID: *rec.UserID, CodeChallenge: rec.CodeChallenge,
		Payload: bundle, ExpiresAt: now.Add(domain.IntegrationCodeTTL), CreatedAt: now,
	}); err != nil {
		return "", domain.Internal()
	}
	return domain.GoogleCallbackURI + "?code=" + url.QueryEscape(integrationCode), nil
}

func (s *GoogleService) Complete(ctx context.Context, userID uuid.UUID, code, codeVerifier string) (*domain.GoogleStatus, error) {
	if !s.cfg.Configured() {
		return nil, domain.NotConfigured("google integration is not configured")
	}
	if s.cipher == nil {
		return nil, domain.NotConfigured("token encryption is not configured")
	}
	rec, err := s.store.GetIntegrationCode(ctx, code)
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
	if rec.UserID != userID {
		return nil, domain.Unauthenticated("code belongs to a different account")
	}
	if codeVerifier == "" || pkceS256(codeVerifier) != rec.CodeChallenge {
		return nil, domain.Unauthenticated("pkce verification failed")
	}
	var bundle struct {
		Ciphertext string `json:"ciphertext"`
		Nonce      string `json:"nonce"`
	}
	if err := json.Unmarshal(rec.Payload, &bundle); err != nil {
		return nil, domain.Internal()
	}
	ct, _ := base64.StdEncoding.DecodeString(bundle.Ciphertext)
	nonce, _ := base64.StdEncoding.DecodeString(bundle.Nonce)
	raw, err := s.cipher.Decrypt(ct, nonce, userID, googleCipherPurpose)
	if err != nil {
		return nil, domain.Internal()
	}
	var payload googleTokenPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, domain.Internal()
	}
	act, an, err := s.cipher.Encrypt(payload.AccessToken, userID, googleCipherPurpose)
	if err != nil {
		return nil, domain.Internal()
	}
	gi := &domain.GoogleIntegration{
		UserID: userID, AccessCiphertext: act, AccessNonce: an,
		Scopes: strings.Fields(payload.Scope),
		CreatedAt: now, UpdatedAt: now,
	}
	if payload.ExpiresIn > 0 {
		exp := now.Add(time.Duration(payload.ExpiresIn) * time.Second)
		gi.TokenExpiresAt = &exp
	}
	if payload.RefreshToken != "" {
		rct, rn, err := s.cipher.Encrypt(payload.RefreshToken, userID, googleCipherPurpose)
		if err != nil {
			return nil, domain.Internal()
		}
		gi.RefreshCiphertext = rct
		gi.RefreshNonce = rn
	}
	err = s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.store.MarkIntegrationCodeUsed(ctx, code, now); err != nil {
			return domain.Unauthenticated("code expired or already used")
		}
		return s.store.PutGoogleIntegration(ctx, gi)
	})
	if err != nil {
		if de, ok := err.(*domain.Error); ok {
			return nil, de
		}
		return nil, domain.Internal()
	}
	return s.Status(ctx, userID)
}

func (s *GoogleService) accessToken(ctx context.Context, gi *domain.GoogleIntegration) (string, error) {
	at, err := s.cipher.Decrypt(gi.AccessCiphertext, gi.AccessNonce, gi.UserID, googleCipherPurpose)
	if err != nil {
		return "", domain.Internal()
	}
	expired := gi.TokenExpiresAt != nil && time.Now().UTC().After(gi.TokenExpiresAt.Add(-time.Minute))
	if !expired || len(gi.RefreshCiphertext) == 0 {
		return at, nil
	}
	rt, err := s.cipher.Decrypt(gi.RefreshCiphertext, gi.RefreshNonce, gi.UserID, googleCipherPurpose)
	if err != nil {
		return "", domain.Internal()
	}
	tokens, err := s.gapi.RefreshAccessToken(ctx, rt)
	if err != nil {
		return "", domain.ProviderError("google token refresh failed")
	}
	ct, nonce, err := s.cipher.Encrypt(tokens.AccessToken, gi.UserID, googleCipherPurpose)
	if err != nil {
		return "", domain.Internal()
	}
	gi.AccessCiphertext = ct
	gi.AccessNonce = nonce
	if tokens.ExpiresIn > 0 {
		exp := time.Now().UTC().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		gi.TokenExpiresAt = &exp
	}
	if err := s.store.PutGoogleIntegration(ctx, gi); err != nil {
		return "", domain.Internal()
	}
	return tokens.AccessToken, nil
}

func (s *GoogleService) Disconnect(ctx context.Context, userID uuid.UUID) error {
	gi, err := s.store.GetGoogleIntegration(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return domain.Internal()
	}
	if at, err := s.cipher.Decrypt(gi.AccessCiphertext, gi.AccessNonce, userID, googleCipherPurpose); err == nil {
		_ = s.gapi.RevokeToken(ctx, at)
	} else if len(gi.RefreshCiphertext) > 0 {
		if rt, err := s.cipher.Decrypt(gi.RefreshCiphertext, gi.RefreshNonce, userID, googleCipherPurpose); err == nil {
			_ = s.gapi.RevokeToken(ctx, rt)
		}
	}
	return s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.store.DeleteGoogleData(ctx, userID); err != nil {
			return err
		}
		return s.store.DeleteGoogleIntegration(ctx, userID)
	})
}

func (s *GoogleService) Sync(ctx context.Context, userID uuid.UUID) (*domain.Operation, error) {
	gi, err := s.store.GetGoogleIntegration(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.IntegrationRequired("google account is not connected")
		}
		return nil, domain.Internal()
	}
	now := time.Now().UTC()
	op := &domain.Operation{
		ID: uuid.New(), UserID: userID, Type: domain.OpGoogleSync,
		Status: domain.OpRunning, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.ops.Create(ctx, op); err != nil {
		return nil, domain.Internal()
	}
	fail := func(code, msg string) (*domain.Operation, error) {
		op.Status = domain.OpFailed
		op.Error = &domain.OperationError{Code: code, Message: msg, Retryable: true}
		op.UpdatedAt = time.Now().UTC()
		_ = s.ops.Update(ctx, op)
		return op, domain.ProviderError(msg)
	}
	token, derr := s.accessToken(ctx, gi)
	if derr != nil {
		return fail(domain.CodeProviderError, "google access token unavailable")
	}
	messages := []domain.GoogleMessageMeta{}
	events := []domain.GoogleEvent{}
	historyID := gi.GmailHistoryID
	syncToken := gi.CalendarSyncToken
	if s.cfg.GmailBeta && gi.HasScope(domain.GoogleScopeGmail) {
		pageToken := ""
		for i := 0; i < 5; i++ {
			ids, next, hid, err := s.gapi.ListMessageIDs(ctx, token, pageToken)
			if err != nil {
				return fail(domain.CodeProviderError, "gmail message list failed")
			}
			for _, id := range ids {
				m, err := s.gapi.GetMessage(ctx, token, id)
				if err != nil {
					return fail(domain.CodeProviderError, "gmail message fetch failed")
				}
				messages = append(messages, *m)
			}
			if hid != "" && hid != "0" {
				historyID = hid
			}
			if next == "" {
				break
			}
			pageToken = next
		}
	}
	if gi.HasScope(domain.GoogleScopeCalendar) {
		evs, nst, err := s.gapi.ListEvents(ctx, token, syncToken)
		if errors.Is(err, domain.ErrSyncTokenInvalid) {
			evs, nst, err = s.gapi.ListEvents(ctx, token, "")
		}
		if err != nil {
			return fail(domain.CodeProviderError, "calendar event list failed")
		}
		events = evs
		if nst != "" {
			syncToken = nst
		}
	}
	gi.GmailHistoryID = historyID
	gi.CalendarSyncToken = syncToken
	gi.LastSyncedAt = &now
	gi.UpdatedAt = now
	err = s.uow.Do(ctx, func(ctx context.Context) error {
		for _, m := range messages {
			if err := s.store.UpsertGoogleMessage(ctx, &domain.GoogleMessage{
				ID: uuid.New(), UserID: userID, ExternalID: m.ID, ThreadID: m.ThreadID,
				Sender: m.From, Subject: m.Subject, Snippet: m.Snippet,
				ReceivedAt: m.ReceivedAt, CreatedAt: now,
			}); err != nil {
				return err
			}
		}
		for _, ev := range events {
			if ev.Status == "cancelled" {
				if err := s.calendar.DeleteExternalCalendarEvent(ctx, userID, ev.ID); err != nil {
					return err
				}
				continue
			}
			tz := ev.TimeZone
			if tz == "" {
				tz = "UTC"
			}
			endsAt := ev.EndsAt
			if !endsAt.After(ev.StartsAt) {
				endsAt = ev.StartsAt.Add(time.Hour)
			}
			extID := ev.ID
			if err := s.calendar.UpsertExternalCalendarEvent(ctx, &domain.CalendarEvent{
				ID: uuid.New(), UserID: userID, Revision: 1, Type: domain.CalCustom,
				Title: ev.Summary, StartsAt: ev.StartsAt, EndsAt: endsAt, TimeZone: tz,
				Source: domain.EventSourceGoogle, ExternalID: &extID,
				CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				return err
			}
		}
		if err := s.store.PutGoogleIntegration(ctx, gi); err != nil {
			return err
		}
		res, _ := json.Marshal(map[string]any{
			"messages": len(messages), "events": len(events), "lastSyncedAt": now,
		})
		op.Status = domain.OpSucceeded
		op.Result = &domain.OperationResult{Kind: string(domain.OpGoogleSync), Value: res}
		op.UpdatedAt = now
		return s.ops.Update(ctx, op)
	})
	if err != nil {
		return fail(domain.CodeInternal, "google sync persistence failed")
	}
	return op, nil
}

func (s *GoogleService) requireConnection(ctx context.Context, userID uuid.UUID) error {
	_, err := s.store.GetGoogleIntegration(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.IntegrationRequired("google account is not connected")
		}
		return domain.Internal()
	}
	return nil
}

func (s *GoogleService) ListMessages(ctx context.Context, userID uuid.UUID, applicationID *uuid.UUID, page domain.PageRequest) (domain.Page[domain.GoogleMessage], error) {
	if err := s.requireConnection(ctx, userID); err != nil {
		return domain.Page[domain.GoogleMessage]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.store.ListGoogleMessages(ctx, userID, applicationID, page, limit+1)
	if err != nil {
		return domain.Page[domain.GoogleMessage]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(m domain.GoogleMessage) domain.Cursor {
		return domain.Cursor{CreatedAt: m.CreatedAt, ID: m.ID}
	}), nil
}

func (s *GoogleService) ListEvents(ctx context.Context, userID uuid.UUID, from, to *time.Time, page domain.PageRequest) (domain.Page[domain.CalendarEvent], error) {
	if err := s.requireConnection(ctx, userID); err != nil {
		return domain.Page[domain.CalendarEvent]{}, err
	}
	src := domain.EventSourceGoogle
	limit := page.EffectiveLimit()
	items, err := s.calendar.ListCalendarEvents(ctx, userID, domain.CalendarEventFilter{
		From: from, To: to, Source: &src, Page: page,
	}, limit+1)
	if err != nil {
		return domain.Page[domain.CalendarEvent]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(e domain.CalendarEvent) domain.Cursor {
		return domain.Cursor{CreatedAt: e.CreatedAt, ID: e.ID}
	}), nil
}

func (s *GoogleService) LinkMessage(ctx context.Context, userID, messageID, applicationID uuid.UUID) (*domain.GoogleMessage, error) {
	if err := s.requireConnection(ctx, userID); err != nil {
		return nil, err
	}
	m, err := s.store.GetGoogleMessage(ctx, userID, messageID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	if _, err := s.applications.Get(ctx, userID, applicationID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	if err := s.store.LinkGoogleMessage(ctx, userID, messageID, applicationID); err != nil {
		return nil, domain.Internal()
	}
	m.ApplicationID = &applicationID
	return m, nil
}
