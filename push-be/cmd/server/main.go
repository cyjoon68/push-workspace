package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
	"push-be/internal/infra"
	"push-be/internal/presentation"
)

func runMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func seedDevUser(ctx context.Context, users domain.UserStore, cfg infra.Config) error {
	if cfg.AppEnv != "development" || cfg.DevAuthToken == "" {
		return nil
	}
	if _, err := users.Get(ctx, cfg.DevUserID); err == nil {
		return nil
	}
	return users.Create(ctx, &domain.User{
		ID: cfg.DevUserID, Provider: "dev", ProviderSubject: cfg.DevUserID.String(),
		DisplayName: "Dev User", Locale: "ko", CreatedAt: time.Now().UTC(),
	})
}

func main() {
	cfg := infra.LoadConfig()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	if err := runMigrations(ctx, pool, "migrations"); err != nil {
		log.Fatal(err)
	}

	db := &infra.DB{Pool: pool}
	users := infra.NewUserRepo(db)
	sessions := infra.NewSessionRepo(db)
	idem := infra.NewIdempotencyRepo(db)
	evidence := infra.NewEvidenceRepo(db)
	sources := infra.NewSourceRepo(db)
	jobs := infra.NewJobRepo(db)
	analyses := infra.NewGapAnalysisRepo(db)
	applications := infra.NewApplicationRepo(db)
	submissions := infra.NewSubmissionRepo(db)
	documents := infra.NewDocumentRepo(db)
	projects := infra.NewProjectRepo(db)
	interviews := infra.NewInterviewRepo(db)
	calendar := infra.NewCalendarRepo(db)
	conversations := infra.NewConversationRepo(db)
	approvals := infra.NewApprovalRepo(db)
	operations := infra.NewOperationRepo(db)
	aiKeys := infra.NewAiKeyRepo(db)
	aiUsage := infra.NewAiUsageRepo(db)
	integrations := infra.NewIntegrationRepo(db)
	billingRepo := infra.NewBillingRepo(db)

	if err := seedDevUser(ctx, users, cfg); err != nil {
		log.Fatal(err)
	}

	providers := map[string]usecase.ProviderConfig{
		"google": {
			ClientID: cfg.Google.ClientID, ClientSecret: cfg.Google.ClientSecret,
			AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:    "https://oauth2.googleapis.com/token",
			UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
		},
		"github": {
			ClientID: cfg.GitHub.ClientID, ClientSecret: cfg.GitHub.ClientSecret,
			AuthURL:     "https://github.com/login/oauth/authorize",
			TokenURL:    "https://github.com/login/oauth/access_token",
			UserInfoURL: "https://api.github.com/user",
		},
	}

	managedAI := cfg.ManagedAIKey != ""
	byokEnabled := cfg.ByokMasterKey != ""
	var keyCipher *infra.KeyCipher
	if byokEnabled {
		keyCipher, err = infra.NewKeyCipher(cfg.ByokMasterKey)
		if err != nil {
			log.Fatal(err)
		}
	}
	var tokenCipher domain.TokenCipher
	if keyCipher != nil {
		tokenCipher = keyCipher
	}
	aiGate := &usecase.AIGate{
		ManagedConfigured: managedAI, ManagedKey: cfg.ManagedAIKey,
		ByokEnabled: byokEnabled, Keys: aiKeys, Cipher: tokenCipher,
		Chat: infra.NewOpenAIClient(),
	}

	authSvc := usecase.NewAuthService(users, sessions, db, infra.NewHTTPOAuthClient(), providers, cfg.AppEnv, cfg.DevAuthToken, cfg.DevUserID)
	approvalSvc := usecase.NewApprovalService(approvals, applications, evidence, documents, submissions, projects, db)
	evidenceSvc := usecase.NewEvidenceService(evidence, operations, db)
	appSvc := usecase.NewApplicationService(applications, jobs, documents, analyses, evidence, interviews, submissions, approvalSvc, db, cfg.JobSiteAdapters)
	jobSvc := usecase.NewJobService(jobs, analyses, applications, evidence, approvalSvc, operations, db, aiGate)
	docSvc := usecase.NewDocumentService(documents, applications, evidence, approvalSvc, operations, db, aiGate)
	projectSvc := usecase.NewProjectService(projects, applications, analyses, approvalSvc, operations, db, aiGate)
	interviewSvc := usecase.NewInterviewService(interviews, applications, jobs, evidence, operations, db, aiGate)
	calendarSvc := usecase.NewCalendarService(calendar, applications)
	conversationSvc := usecase.NewConversationService(conversations, applications, documents, evidence, operations, db, aiGate, aiUsage)
	aiSvc := usecase.NewAIService(aiGate, operations, aiUsage, applications, db)
	opSvc := usecase.NewOperationService(operations, db, evidenceSvc)
	googleSvc := usecase.NewGoogleService(integrations, sessions, calendar, applications, operations, db, tokenCipher,
		infra.NewGoogleClient(cfg.Google.ClientID, cfg.Google.ClientSecret),
		usecase.GoogleConfig{
			ClientID: cfg.Google.ClientID, ClientSecret: cfg.Google.ClientSecret,
			AuthURL:   "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:  "https://oauth2.googleapis.com/token",
			GmailBeta: cfg.GmailBeta,
		})
	billingSvc := usecase.NewBillingService(billingRepo, infra.NewStripeClient(cfg.StripeSecret), db,
		usecase.BillingConfig{
			Configured:    cfg.StripeSecret != "",
			Prices:        map[string]string{domain.PlanPro: cfg.StripePricePro, domain.PlanUltra: cfg.StripePriceUltra},
			SuccessURL:    cfg.StripeSuccessURL,
			CancelURL:     cfg.StripeCancelURL,
			ReturnURL:     cfg.StripePortalURL,
			WebhookSecret: cfg.StripeWebhookSecret,
		})

	srv := presentation.NewServer(presentation.Deps{
		Auth: authSvc, Evidence: evidenceSvc, Jobs: jobSvc, Applications: appSvc,
		Documents: docSvc, Projects: projectSvc, Interviews: interviewSvc, Calendar: calendarSvc,
		Approvals: approvalSvc, Operations: opSvc, Sources: sources, AiKeys: aiKeys,
		AI:            aiSvc,
		Google:        googleSvc,
		Billing:       billingSvc,
		Conversations: conversationSvc,
		Idempotency:   idem,
		Config: presentation.ConfigView{
			GoogleConfigured: cfg.Google.ClientID != "" && cfg.Google.ClientSecret != "",
			GmailBeta:        cfg.GmailBeta,
			ManagedAI:        managedAI,
			ByokEnabled:      byokEnabled,
			StripeConfigured: cfg.StripeSecret != "",
			JobSiteAdapters:  cfg.JobSiteAdapters,
		},
		StorageDir: cfg.StorageDir,
		MasterKey:  cfg.ByokMasterKey,
	})

	go func() {
		if err := srv.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
