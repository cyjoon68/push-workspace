package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"push-be/internal/domain/usecase"
)

type HTTPOAuthClient struct {
	http *http.Client
}

func NewHTTPOAuthClient() *HTTPOAuthClient {
	return &HTTPOAuthClient{http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *HTTPOAuthClient) ExchangeCode(ctx context.Context, cfg usecase.ProviderConfig, code, redirectURI string) (string, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK || body.AccessToken == "" {
		return "", fmt.Errorf("token exchange failed: status %d %s", resp.StatusCode, body.Error)
	}
	return body.AccessToken, nil
}

func (c *HTTPOAuthClient) FetchIdentity(ctx context.Context, cfg usecase.ProviderConfig, accessToken string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.UserInfoURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", errors.New("identity fetch failed")
	}
	var body struct {
		Sub   string `json:"sub"`
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", err
	}
	subject := body.Sub
	if subject == "" && body.ID != 0 {
		subject = strconv.FormatInt(body.ID, 10)
	}
	if subject == "" {
		return "", "", errors.New("missing subject")
	}
	name := body.Name
	if name == "" {
		name = body.Login
	}
	if name == "" {
		name = body.Email
	}
	if name == "" {
		name = "user"
	}
	return subject, name, nil
}
