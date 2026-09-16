package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const stripeWebhookTolerance = 5 * time.Minute

type BillingConfig struct {
	Configured    bool
	Prices        map[string]string
	SuccessURL    string
	CancelURL     string
	ReturnURL     string
	WebhookSecret string
}

type BillingService struct {
	billing domain.BillingStore
	stripe  domain.StripeGateway
	uow     domain.UnitOfWork
	cfg     BillingConfig
}

func NewBillingService(billing domain.BillingStore, stripe domain.StripeGateway, uow domain.UnitOfWork, cfg BillingConfig) *BillingService {
	return &BillingService{billing: billing, stripe: stripe, uow: uow, cfg: cfg}
}

func (s *BillingService) Status(ctx context.Context, u *domain.User) (*domain.BillingStatus, error) {
	bal, err := s.billing.LastBalance(ctx, u.ID)
	if err != nil {
		return nil, domain.Internal()
	}
	st := &domain.BillingStatus{
		SubscriptionStatus: u.SubscriptionStatus,
		PeriodEndsAt:       u.PeriodEndsAt,
		BalanceMicroCredits: bal,
	}
	if st.SubscriptionStatus == "" {
		st.SubscriptionStatus = domain.SubStatusNone
	}
	if u.Plan != "" {
		plan := u.Plan
		st.Plan = &plan
	}
	return st, nil
}

func (s *BillingService) Checkout(ctx context.Context, u *domain.User, planID string) (*domain.StripeCheckout, error) {
	if !s.cfg.Configured {
		return nil, domain.NotConfigured("billing is not configured")
	}
	if planID == domain.PlanFree || !domain.ValidPlan(planID) {
		return nil, domain.ValidationField("planId", "unsupported plan")
	}
	price := s.cfg.Prices[planID]
	if price == "" {
		return nil, domain.NotConfigured("plan price is not configured")
	}
	customerID := ""
	if u.StripeCustomerID != nil {
		customerID = *u.StripeCustomerID
	}
	sess, err := s.stripe.CreateCheckoutSession(ctx, price, customerID, u.ID.String(), planID, s.cfg.SuccessURL, s.cfg.CancelURL)
	if err != nil {
		return nil, domain.ProviderError("stripe checkout session failed")
	}
	return sess, nil
}

func (s *BillingService) Portal(ctx context.Context, u *domain.User) (string, error) {
	if !s.cfg.Configured {
		return "", domain.NotConfigured("billing is not configured")
	}
	if u.StripeCustomerID == nil || *u.StripeCustomerID == "" {
		return "", domain.InvalidTransition("no billing account")
	}
	url, err := s.stripe.CreatePortalSession(ctx, *u.StripeCustomerID, s.cfg.ReturnURL)
	if err != nil {
		return "", domain.ProviderError("stripe portal session failed")
	}
	return url, nil
}

func (s *BillingService) Ledger(ctx context.Context, userID uuid.UUID, page domain.PageRequest) (domain.Page[domain.LedgerEntry], error) {
	limit := page.EffectiveLimit()
	items, err := s.billing.ListLedger(ctx, userID, page, limit+1)
	if err != nil {
		return domain.Page[domain.LedgerEntry]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(e domain.LedgerEntry) domain.Cursor {
		return domain.Cursor{CreatedAt: e.CreatedAt, ID: e.ID}
	}), nil
}

func VerifyStripeSignature(payload []byte, header, secret string, now time.Time) error {
	var ts int64
	sigs := []string{}
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			v, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return errors.New("bad timestamp")
			}
			ts = v
		case "v1":
			sigs = append(sigs, kv[1])
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return errors.New("missing signature")
	}
	if d := now.Unix() - ts; d > int64(stripeWebhookTolerance.Seconds()) || d < -int64(stripeWebhookTolerance.Seconds()) {
		return errors.New("stale timestamp")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.", ts)
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, sig := range sigs {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}
	return errors.New("signature mismatch")
}

func (s *BillingService) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	if !s.cfg.Configured || s.cfg.WebhookSecret == "" {
		return domain.NotConfigured("billing is not configured")
	}
	if err := VerifyStripeSignature(payload, sigHeader, s.cfg.WebhookSecret, time.Now()); err != nil {
		return domain.Validation("invalid webhook signature")
	}
	var event struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &event); err != nil || event.ID == "" {
		return domain.Validation("invalid webhook payload")
	}
	fresh, err := s.billing.RecordStripeEvent(ctx, event.ID, event.Type, time.Now().UTC())
	if err != nil {
		return domain.Internal()
	}
	if !fresh {
		return nil
	}
	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event.Data.Object)
	case "customer.subscription.updated", "customer.subscription.deleted":
		return s.handleSubscriptionChanged(ctx, event.Type, event.Data.Object)
	default:
		return nil
	}
}

func (s *BillingService) handleCheckoutCompleted(ctx context.Context, raw json.RawMessage) error {
	var obj struct {
		ID                string `json:"id"`
		ClientReferenceID string `json:"client_reference_id"`
		Customer          string `json:"customer"`
		Metadata          struct {
			PlanID string `json:"planId"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return domain.Validation("invalid session payload")
	}
	userID, err := uuid.Parse(obj.ClientReferenceID)
	if err != nil {
		return domain.Validation("invalid session reference")
	}
	planID := obj.Metadata.PlanID
	if !domain.ValidPlan(planID) || planID == domain.PlanFree {
		return domain.Validation("unsupported plan")
	}
	credits := domain.PlanCreditsMicroCredits[planID]
	return s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.billing.UpdateSubscription(ctx, userID, planID, domain.SubStatusActive, &obj.Customer, nil); err != nil {
			return err
		}
		bal, err := s.billing.LastBalance(ctx, userID)
		if err != nil {
			return err
		}
		return s.billing.AppendLedger(ctx, &domain.LedgerEntry{
			ID: uuid.New(), UserID: userID, Type: domain.LedgerPurchase,
			AmountMicroCredits: credits, BalanceAfter: bal + credits,
			ReferenceID: &obj.ID, CreatedAt: time.Now().UTC(),
		})
	})
}

func (s *BillingService) handleSubscriptionChanged(ctx context.Context, typ string, raw json.RawMessage) error {
	var obj struct {
		Customer           string `json:"customer"`
		Status             string `json:"status"`
		CurrentPeriodEnd   int64  `json:"current_period_end"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil || obj.Customer == "" {
		return domain.Validation("invalid subscription payload")
	}
	var periodEnd *time.Time
	if obj.CurrentPeriodEnd > 0 {
		t := time.Unix(obj.CurrentPeriodEnd, 0).UTC()
		periodEnd = &t
	}
	status := domain.SubStatusActive
	if typ == "customer.subscription.deleted" || obj.Status == "canceled" {
		status = domain.SubStatusCanceled
	}
	if err := s.billing.UpdateSubscriptionByCustomer(ctx, obj.Customer, status, periodEnd); err != nil {
		return domain.Internal()
	}
	return nil
}
