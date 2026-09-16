package infra

import (
	"os"

	"github.com/google/uuid"
)

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
}

type Config struct {
	Port                string
	AppEnv              string
	DatabaseURL         string
	DevAuthToken        string
	DevUserID           uuid.UUID
	Google              OAuthProviderConfig
	GitHub              OAuthProviderConfig
	ManagedAIKey        string
	ByokMasterKey       string
	StripeSecret        string
	StripeWebhookSecret string
	StripePricePro      string
	StripePriceUltra    string
	StripeSuccessURL    string
	StripeCancelURL     string
	StripePortalURL     string
	StorageDir          string
	GmailBeta           bool
	JobSiteAdapters     map[string]bool
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func LoadConfig() Config {
	devUser, err := uuid.Parse(env("DEV_USER_ID", "00000000-0000-4000-8000-000000000001"))
	if err != nil {
		devUser = uuid.MustParse("00000000-0000-4000-8000-000000000001")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://" + env("PGUSER", "push") + ":" + env("PGPASSWORD", "push") +
			"@" + env("PGHOST", "localhost") + ":" + env("PGPORT", "5432") + "/" + env("PGDATABASE", "push") + "?sslmode=disable"
	}
	return Config{
		Port:         env("PORT", "8080"),
		AppEnv:       env("APP_ENV", "development"),
		DatabaseURL:  dbURL,
		DevAuthToken: os.Getenv("DEV_AUTH_TOKEN"),
		DevUserID:    devUser,
		Google: OAuthProviderConfig{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		},
		GitHub: OAuthProviderConfig{
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		},
		ManagedAIKey:        os.Getenv("OPENAI_API_KEY"),
		ByokMasterKey:       os.Getenv("BYOK_MASTER_KEY"),
		StripeSecret:        os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripePricePro:      os.Getenv("STRIPE_PRICE_PRO"),
		StripePriceUltra:    os.Getenv("STRIPE_PRICE_ULTRA"),
		StripeSuccessURL:    env("STRIPE_SUCCESS_URL", "push://billing/success"),
		StripeCancelURL:     env("STRIPE_CANCEL_URL", "push://billing/cancel"),
		StripePortalURL:     env("STRIPE_PORTAL_RETURN_URL", "push://billing/portal"),
		StorageDir:          env("STORAGE_DIR", "./data/sources"),
		GmailBeta:           os.Getenv("GOOGLE_GMAIL_BETA_ENABLED") == "true",
		JobSiteAdapters:     map[string]bool{},
	}
}
