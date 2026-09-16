package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                 uuid.UUID  `json:"id"`
	Provider           string     `json:"-"`
	ProviderSubject    string     `json:"-"`
	DisplayName        string     `json:"displayName"`
	Locale             string     `json:"locale"`
	Plan               string     `json:"-"`
	SubscriptionStatus string     `json:"-"`
	StripeCustomerID   *string    `json:"-"`
	PeriodEndsAt       *time.Time `json:"-"`
	CreatedAt          time.Time  `json:"createdAt"`
}

type OAuthState struct {
	State         string
	Provider      string
	Purpose       string
	UserID        *uuid.UUID
	CodeChallenge string
	RedirectURI   string
	ExpiresAt     time.Time
	CreatedAt     time.Time
}

type ExchangeCode struct {
	Code          string
	UserID        uuid.UUID
	CodeChallenge string
	ExpiresAt     time.Time
	UsedAt        *time.Time
	CreatedAt     time.Time
}

type AccessToken struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	TokenHash      string
	RefreshTokenID uuid.UUID
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type Session struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         *User
}

const (
	AccessTokenTTLSeconds = 900
	RefreshTokenTTL       = 30 * 24 * time.Hour
	OAuthStateTTL         = 10 * time.Minute
	ExchangeCodeTTL       = 60 * time.Second
	IntegrationCodeTTL    = 60 * time.Second
	AuthCallbackURI       = "push://auth/callback"
	GoogleCallbackURI     = "push://integrations/google/callback"
)

const (
	OAuthPurposeLogin  = "LOGIN"
	OAuthPurposeGoogle = "GOOGLE"
)

func ValidOAuthProvider(p string) bool {
	return p == "google" || p == "github"
}
