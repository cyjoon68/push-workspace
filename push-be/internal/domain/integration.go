package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrSyncTokenInvalid = errors.New("google sync token invalid")

const (
	GoogleScopeGmail    = "https://www.googleapis.com/auth/gmail.readonly"
	GoogleScopeCalendar = "https://www.googleapis.com/auth/calendar.events"
)

type IntegrationCode struct {
	Code          string
	UserID        uuid.UUID
	CodeChallenge string
	Payload       []byte
	ExpiresAt     time.Time
	UsedAt        *time.Time
	CreatedAt     time.Time
}

type GoogleIntegration struct {
	UserID            uuid.UUID
	AccessCiphertext  []byte
	AccessNonce       []byte
	RefreshCiphertext []byte
	RefreshNonce      []byte
	Scopes            []string
	TokenExpiresAt    *time.Time
	GmailHistoryID    string
	CalendarSyncToken string
	LastSyncedAt      *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (g *GoogleIntegration) HasScope(scope string) bool {
	for _, s := range g.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

type GoogleMessage struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"-"`
	ExternalID    string     `json:"-"`
	ThreadID      string     `json:"threadId"`
	ApplicationID *uuid.UUID `json:"applicationId"`
	Sender        string     `json:"sender"`
	Subject       string     `json:"subject"`
	Snippet       string     `json:"snippet"`
	ReceivedAt    *time.Time `json:"receivedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type GoogleTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	Scope        string
}

type GoogleMessageMeta struct {
	ID         string
	ThreadID   string
	From       string
	Subject    string
	Snippet    string
	ReceivedAt *time.Time
}

type GoogleEvent struct {
	ID       string
	Status   string
	Summary  string
	StartsAt time.Time
	EndsAt   time.Time
	TimeZone string
}

type GoogleStatus struct {
	Enabled        bool       `json:"enabled"`
	Connected      bool       `json:"connected"`
	Scopes         []string   `json:"scopes"`
	GmailStatus    string     `json:"gmailStatus"`
	CalendarStatus string     `json:"calendarStatus"`
	LastSyncedAt   *time.Time `json:"lastSyncedAt"`
}

type LedgerEntry struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"-"`
	Type               string    `json:"type"`
	AmountMicroCredits int64     `json:"amountMicroCredits"`
	BalanceAfter       int64     `json:"balanceAfter"`
	ReferenceID        *string   `json:"referenceId"`
	CreatedAt          time.Time `json:"createdAt"`
}

type BillingStatus struct {
	SubscriptionStatus   string     `json:"subscriptionStatus"`
	Plan                 *string    `json:"plan"`
	PeriodEndsAt         *time.Time `json:"periodEndsAt"`
	BalanceMicroCredits  int64      `json:"balanceMicroCredits"`
	ReservedMicroCredits int64      `json:"reservedMicroCredits"`
}
