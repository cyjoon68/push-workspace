package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

func signStripe(t *testing.T, payload []byte, secret string, ts time.Time) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.", ts.Unix())
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyStripeSignature(t *testing.T) {
	payload := []byte(`{"id":"evt_1","type":"checkout.session.completed"}`)
	secret := "whsec_test"
	header := signStripe(t, payload, secret, time.Now())
	if err := VerifyStripeSignature(payload, header, secret, time.Now()); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
}

func TestVerifyStripeSignatureBad(t *testing.T) {
	payload := []byte(`{}`)
	secret := "whsec_test"
	header := signStripe(t, []byte(`{"other":1}`), secret, time.Now())
	if err := VerifyStripeSignature(payload, header, secret, time.Now()); err == nil {
		t.Fatal("expected signature mismatch")
	}
}

func TestVerifyStripeSignatureStale(t *testing.T) {
	payload := []byte(`{}`)
	secret := "whsec_test"
	header := signStripe(t, payload, secret, time.Now().Add(-10*time.Minute))
	if err := VerifyStripeSignature(payload, header, secret, time.Now()); err == nil {
		t.Fatal("expected stale timestamp rejection")
	}
}

type stubBillingStore struct {
	user      *domain.User
	balance   int64
	ledger    []domain.LedgerEntry
	eventSeen bool
}

func (s *stubBillingStore) RecordStripeEvent(ctx context.Context, eventID, typ string, at time.Time) (bool, error) {
	return !s.eventSeen, nil
}
func (s *stubBillingStore) UpdateSubscription(ctx context.Context, userID uuid.UUID, plan, status string, customerID *string, periodEndsAt *time.Time) error {
	return nil
}
func (s *stubBillingStore) UpdateSubscriptionByCustomer(ctx context.Context, customerID, status string, periodEndsAt *time.Time) error {
	return nil
}
func (s *stubBillingStore) UserIDByStripeCustomer(ctx context.Context, customerID string) (uuid.UUID, error) {
	return uuid.Nil, domain.ErrNotFound
}
func (s *stubBillingStore) AppendLedger(ctx context.Context, e *domain.LedgerEntry) error {
	s.ledger = append(s.ledger, *e)
	return nil
}
func (s *stubBillingStore) ListLedger(ctx context.Context, userID uuid.UUID, page domain.PageRequest, limit int) ([]domain.LedgerEntry, error) {
	return s.ledger, nil
}
func (s *stubBillingStore) LastBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.balance, nil
}

type stubStripeGateway struct {
	checkout *domain.StripeCheckout
	err      error
}

func (s *stubStripeGateway) CreateCheckoutSession(ctx context.Context, priceID, customerID, userID, planID, successURL, cancelURL string) (*domain.StripeCheckout, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.checkout, nil
}
func (s *stubStripeGateway) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return "https://billing.stripe.com/session/x", nil
}

func newBillingService(b domain.BillingStore, gw domain.StripeGateway, prices map[string]string) *BillingService {
	return NewBillingService(b, gw, noopUoW{}, BillingConfig{
		Configured: true, Prices: prices,
		SuccessURL: "https://app/ok", CancelURL: "https://app/cancel", ReturnURL: "https://app",
	})
}

func TestCheckoutRejectsUnknownPlan(t *testing.T) {
	svc := newBillingService(&stubBillingStore{}, &stubStripeGateway{}, map[string]string{domain.PlanUltra: "price_1"})
	_, err := svc.Checkout(context.Background(), &domain.User{ID: uuid.New()}, "ENTERPRISE")
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}

func TestCheckoutNotConfiguredWithoutPrice(t *testing.T) {
	svc := newBillingService(&stubBillingStore{}, &stubStripeGateway{}, map[string]string{})
	_, err := svc.Checkout(context.Background(), &domain.User{ID: uuid.New()}, domain.PlanUltra)
	if errCode(t, err) != domain.CodeNotConfigured {
		t.Fatalf("want NOT_CONFIGURED, got %v", err)
	}
}

func TestCheckoutReturnsSessionURL(t *testing.T) {
	exp := time.Now().Add(time.Hour).UTC()
	gw := &stubStripeGateway{checkout: &domain.StripeCheckout{URL: "https://checkout.stripe.com/x", ExpiresAt: &exp}}
	svc := newBillingService(&stubBillingStore{}, gw, map[string]string{domain.PlanUltra: "price_1"})
	out, err := svc.Checkout(context.Background(), &domain.User{ID: uuid.New()}, domain.PlanUltra)
	if err != nil {
		t.Fatal(err)
	}
	if out.URL != "https://checkout.stripe.com/x" {
		t.Fatalf("got %q", out.URL)
	}
}

func TestWebhookRejectsBadSignature(t *testing.T) {
	svc := newBillingService(&stubBillingStore{}, &stubStripeGateway{}, nil)
	svc.cfg.WebhookSecret = "whsec_test"
	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "t=1,v1=bad")
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}

func TestWebhookCheckoutCompletedGrantsCredits(t *testing.T) {
	userID := uuid.New()
	secret := "whsec_test"
	payload := []byte(fmt.Sprintf(
		`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_1","client_reference_id":"%s","customer":"cus_1","metadata":{"planId":"ULTRA"}}}}`, userID))
	store := &stubBillingStore{}
	svc := newBillingService(store, &stubStripeGateway{}, nil)
	svc.cfg.WebhookSecret = secret
	header := signStripe(t, payload, secret, time.Now())
	if err := svc.HandleWebhook(context.Background(), payload, header); err != nil {
		t.Fatal(err)
	}
	if len(store.ledger) != 1 || store.ledger[0].AmountMicroCredits != domain.PlanCreditsMicroCredits[domain.PlanUltra] {
		t.Fatalf("ledger = %+v", store.ledger)
	}
}
