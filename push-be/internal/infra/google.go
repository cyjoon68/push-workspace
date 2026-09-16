package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"push-be/internal/domain"
)

type GoogleClient struct {
	http         *http.Client
	clientID     string
	clientSecret string
}

func NewGoogleClient(clientID, clientSecret string) *GoogleClient {
	return &GoogleClient{
		http:         &http.Client{Timeout: 20 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (c *GoogleClient) tokenRequest(ctx context.Context, form url.Values) (*domain.GoogleTokens, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK || body.AccessToken == "" {
		return nil, fmt.Errorf("google token request failed: status %d %s", resp.StatusCode, body.Error)
	}
	return &domain.GoogleTokens{
		AccessToken: body.AccessToken, RefreshToken: body.RefreshToken,
		ExpiresIn: body.ExpiresIn, Scope: body.Scope,
	}, nil
}

func (c *GoogleClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*domain.GoogleTokens, error) {
	form := url.Values{}
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	return c.tokenRequest(ctx, form)
}

func (c *GoogleClient) RefreshAccessToken(ctx context.Context, refreshToken string) (*domain.GoogleTokens, error) {
	form := url.Values{}
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")
	return c.tokenRequest(ctx, form)
}

func (c *GoogleClient) RevokeToken(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/revoke?token="+url.QueryEscape(token), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *GoogleClient) getJSON(ctx context.Context, accessToken, rawURL string, dst any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return resp.StatusCode, err
	}
	return resp.StatusCode, nil
}

func (c *GoogleClient) ListMessageIDs(ctx context.Context, accessToken, pageToken string) ([]string, string, string, error) {
	u := "https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=50"
	if pageToken != "" {
		u += "&pageToken=" + url.QueryEscape(pageToken)
	}
	var body struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
		NextPageToken string `json:"nextPageToken"`
		HistoryID     uint64 `json:"historyId"`
	}
	status, err := c.getJSON(ctx, accessToken, u, &body)
	if err != nil {
		return nil, "", "", err
	}
	if status != http.StatusOK {
		return nil, "", "", fmt.Errorf("gmail list failed: status %d", status)
	}
	ids := []string{}
	for _, m := range body.Messages {
		ids = append(ids, m.ID)
	}
	return ids, body.NextPageToken, strconv.FormatUint(body.HistoryID, 10), nil
}

func (c *GoogleClient) GetMessage(ctx context.Context, accessToken, id string) (*domain.GoogleMessageMeta, error) {
	u := "https://gmail.googleapis.com/gmail/v1/users/me/messages/" + url.PathEscape(id) +
		"?format=metadata&metadataHeaders=From&metadataHeaders=Subject&metadataHeaders=Date"
	var body struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
		Snippet  string `json:"snippet"`
		Payload  struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"payload"`
		InternalDate string `json:"internalDate"`
	}
	status, err := c.getJSON(ctx, accessToken, u, &body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("gmail get failed: status %d", status)
	}
	m := &domain.GoogleMessageMeta{ID: body.ID, ThreadID: body.ThreadID, Snippet: body.Snippet}
	for _, h := range body.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from":
			m.From = h.Value
		case "subject":
			m.Subject = h.Value
		}
	}
	if ms, err := strconv.ParseInt(body.InternalDate, 10, 64); err == nil {
		t := time.UnixMilli(ms).UTC()
		m.ReceivedAt = &t
	}
	return m, nil
}

func (c *GoogleClient) ListEvents(ctx context.Context, accessToken, syncToken string) ([]domain.GoogleEvent, string, error) {
	u := "https://www.googleapis.com/calendar/v3/calendars/primary/events?maxResults=100"
	if syncToken != "" {
		u += "&syncToken=" + url.QueryEscape(syncToken)
	} else {
		u += "&singleEvents=true&timeMin=" + url.QueryEscape(time.Now().UTC().Add(-90*24*time.Hour).Format(time.RFC3339))
	}
	var body struct {
		Items []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
			Start   struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
				TimeZone string `json:"timeZone"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"end"`
		} `json:"items"`
		NextPageToken string `json:"nextPageToken"`
		NextSyncToken string `json:"nextSyncToken"`
	}
	status, err := c.getJSON(ctx, accessToken, u, &body)
	if err != nil {
		return nil, "", err
	}
	if status == http.StatusGone {
		return nil, "", domain.ErrSyncTokenInvalid
	}
	if status != http.StatusOK {
		return nil, "", fmt.Errorf("calendar list failed: status %d", status)
	}
	events := []domain.GoogleEvent{}
	for _, it := range body.Items {
		e := domain.GoogleEvent{ID: it.ID, Status: it.Status, Summary: it.Summary, TimeZone: it.Start.TimeZone}
		e.StartsAt = parseGoogleTime(it.Start.DateTime, it.Start.Date)
		e.EndsAt = parseGoogleTime(it.End.DateTime, it.End.Date)
		events = append(events, e)
	}
	return events, body.NextSyncToken, nil
}

func parseGoogleTime(dt, d string) time.Time {
	if dt != "" {
		if t, err := time.Parse(time.RFC3339, dt); err == nil {
			return t
		}
	}
	if d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			return t
		}
	}
	return time.Time{}
}
