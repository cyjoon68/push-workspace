package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"push-be/internal/domain"
)

type StripeClient struct {
	http   *http.Client
	secret string
}

func NewStripeClient(secret string) *StripeClient {
	return &StripeClient{http: &http.Client{Timeout: 20 * time.Second}, secret: secret}
}

func (c *StripeClient) post(ctx context.Context, path string, form url.Values, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.stripe.com"+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.secret, "")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var apiErr struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if resp.StatusCode != http.StatusOK {
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("stripe %s failed: status %d %s", path, resp.StatusCode, apiErr.Error.Message)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func (c *StripeClient) CreateCheckoutSession(ctx context.Context, priceID, customerID, userID, planID, successURL, cancelURL string) (*domain.StripeCheckout, error) {
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("line_items[0][price]", priceID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	form.Set("client_reference_id", userID)
	form.Set("metadata[planId]", planID)
	if customerID != "" {
		form.Set("customer", customerID)
	}
	var body struct {
		URL       string `json:"url"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := c.post(ctx, "/v1/checkout/sessions", form, &body); err != nil {
		return nil, err
	}
	if body.URL == "" {
		return nil, fmt.Errorf("stripe checkout session missing url")
	}
	out := &domain.StripeCheckout{URL: body.URL}
	if body.ExpiresAt > 0 {
		t := time.Unix(body.ExpiresAt, 0).UTC()
		out.ExpiresAt = &t
	}
	return out, nil
}

func (c *StripeClient) CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	form := url.Values{}
	form.Set("customer", customerID)
	form.Set("return_url", returnURL)
	var body struct {
		URL string `json:"url"`
	}
	if err := c.post(ctx, "/v1/billing_portal/sessions", form, &body); err != nil {
		return "", err
	}
	if body.URL == "" {
		return "", fmt.Errorf("stripe portal session missing url")
	}
	return body.URL, nil
}
