package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type UserStore interface {
	Create(ctx context.Context, u *User) error
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetByProvider(ctx context.Context, provider, subject string) (*User, error)
}

type SessionStore interface {
	SaveOAuthState(ctx context.Context, s *OAuthState) error
	GetOAuthState(ctx context.Context, state string) (*OAuthState, error)
	DeleteOAuthState(ctx context.Context, state string) error
	SaveExchangeCode(ctx context.Context, c *ExchangeCode) error
	GetExchangeCode(ctx context.Context, code string) (*ExchangeCode, error)
	MarkExchangeCodeUsed(ctx context.Context, code string, at time.Time) error
	SaveAccessToken(ctx context.Context, t *AccessToken) error
	GetAccessToken(ctx context.Context, tokenHash string) (*AccessToken, error)
	SaveRefreshToken(ctx context.Context, t *RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeAccessTokensForRefresh(ctx context.Context, refreshTokenID uuid.UUID, at time.Time) error
}

type IdempotencyRecord struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Key            uuid.UUID
	Method         string
	Path           string
	RequestHash    string
	ResponseStatus *int
	ResponseBody   []byte
	CreatedAt      time.Time
}

type IdempotencyStore interface {
	InsertPending(ctx context.Context, rec *IdempotencyRecord) (bool, error)
	Get(ctx context.Context, userID uuid.UUID, method, path string, key uuid.UUID) (*IdempotencyRecord, error)
	Complete(ctx context.Context, id uuid.UUID, status int, body []byte) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type EvidenceFilter struct {
	Kind  *EvidenceKind
	Query string
	Page  PageRequest
}

type EvidenceStore interface {
	Create(ctx context.Context, e *CareerEvidence) error
	Get(ctx context.Context, userID, id uuid.UUID) (*CareerEvidence, error)
	List(ctx context.Context, userID uuid.UUID, f EvidenceFilter, limit int) ([]CareerEvidence, error)
	Update(ctx context.Context, e *CareerEvidence, expectedRevision int64) error
	Count(ctx context.Context, userID uuid.UUID) (int, error)
}

type JobFilter struct {
	Query    string
	Archived *bool
	Page     PageRequest
}

type JobStore interface {
	Create(ctx context.Context, j *JobPosting) error
	Get(ctx context.Context, userID, id uuid.UUID) (*JobPosting, error)
	List(ctx context.Context, userID uuid.UUID, f JobFilter, limit int) ([]JobPosting, error)
	Update(ctx context.Context, j *JobPosting, expectedRevision int64) error
}

type GapAnalysisStore interface {
	Create(ctx context.Context, g *GapAnalysis) error
	Get(ctx context.Context, userID, id uuid.UUID) (*GapAnalysis, error)
	ListByJob(ctx context.Context, userID, jobID uuid.UUID, page PageRequest, limit int) ([]GapAnalysis, error)
	MarkStaleForJob(ctx context.Context, userID, jobID uuid.UUID, exceptID uuid.UUID) error
	CountByApplication(ctx context.Context, userID, applicationID uuid.UUID) (int, error)
}

type ApplicationFilter struct {
	Stage *ApplicationStage
	Query string
	Page  PageRequest
}

type ApplicationStore interface {
	Create(ctx context.Context, a *Application) error
	Get(ctx context.Context, userID, id uuid.UUID) (*Application, error)
	List(ctx context.Context, userID uuid.UUID, f ApplicationFilter, limit int) ([]Application, error)
	Update(ctx context.Context, a *Application, expectedRevision int64) error
	AddEvent(ctx context.Context, e *ApplicationEvent) error
	ListEvents(ctx context.Context, userID, applicationID uuid.UUID, page PageRequest, limit int) ([]ApplicationEvent, error)
}

type SubmissionStore interface {
	CreateDraft(ctx context.Context, d *SubmissionDraft) error
	GetDraft(ctx context.Context, userID, id uuid.UUID) (*SubmissionDraft, error)
	Create(ctx context.Context, s *Submission) error
	List(ctx context.Context, userID, applicationID uuid.UUID, page PageRequest, limit int) ([]Submission, error)
}

type DocumentFilter struct {
	ApplicationID *uuid.UUID
	Kind          *DocumentKind
	Page          PageRequest
}

type DocumentStore interface {
	Create(ctx context.Context, d *Document) error
	Get(ctx context.Context, userID, id uuid.UUID) (*Document, error)
	List(ctx context.Context, userID uuid.UUID, f DocumentFilter, limit int) ([]Document, error)
	Update(ctx context.Context, d *Document, expectedRevision int64) error
	CreateVersion(ctx context.Context, v *DocumentVersion) error
	GetVersion(ctx context.Context, userID, documentID, versionID uuid.UUID) (*DocumentVersion, error)
	GetVersionByID(ctx context.Context, userID, versionID uuid.UUID) (*DocumentVersion, error)
	ListVersions(ctx context.Context, userID, documentID uuid.UUID, page PageRequest, limit int) ([]DocumentVersion, error)
	NextVersionNumber(ctx context.Context, documentID uuid.UUID) (int, error)
	CreateProposal(ctx context.Context, p *RevisionProposal) error
	GetProposal(ctx context.Context, userID, documentID, proposalID uuid.UUID) (*RevisionProposal, error)
	MarkProposalApplied(ctx context.Context, id uuid.UUID, at time.Time) error
	CreateExport(ctx context.Context, e *DocumentExport) error
	GetExport(ctx context.Context, userID, documentID, exportID uuid.UUID) (*DocumentExport, error)
	ListExports(ctx context.Context, userID, documentID uuid.UUID, page PageRequest, limit int) ([]DocumentExport, error)
	UpdateExportResult(ctx context.Context, e *DocumentExport) error
	CountFinalizedByApplication(ctx context.Context, userID, applicationID uuid.UUID) (int, error)
}

type ProjectFilter struct {
	ApplicationID *uuid.UUID
	Page          PageRequest
}

type ProjectStore interface {
	CreateBlueprint(ctx context.Context, b *ProjectBlueprint) error
	GetBlueprint(ctx context.Context, userID, id uuid.UUID) (*ProjectBlueprint, error)
	ListBlueprints(ctx context.Context, userID uuid.UUID, f ProjectFilter, limit int) ([]ProjectBlueprint, error)
	UpdateBlueprint(ctx context.Context, b *ProjectBlueprint, expectedRevision int64) error
	CreateRun(ctx context.Context, r *CliRun) error
	GetRun(ctx context.Context, userID, projectID, runID uuid.UUID) (*CliRun, error)
	ListRuns(ctx context.Context, userID, projectID uuid.UUID, page PageRequest, limit int) ([]CliRun, error)
	UpdateRun(ctx context.Context, r *CliRun, expectedRevision int64) error
	CreateEvidence(ctx context.Context, e *ProjectEvidence) error
	GetEvidence(ctx context.Context, userID, projectID, evidenceID uuid.UUID) (*ProjectEvidence, error)
	ListEvidence(ctx context.Context, userID, projectID uuid.UUID, page PageRequest, limit int) ([]ProjectEvidence, error)
	UpdateEvidence(ctx context.Context, e *ProjectEvidence, expectedRevision int64) error
}

type InterviewFilter struct {
	ApplicationID *uuid.UUID
	From          *time.Time
	To            *time.Time
	Page          PageRequest
}

type InterviewStore interface {
	Create(ctx context.Context, s *InterviewSession) error
	Get(ctx context.Context, userID, id uuid.UUID) (*InterviewSession, error)
	List(ctx context.Context, userID uuid.UUID, f InterviewFilter, limit int) ([]InterviewSession, error)
	Update(ctx context.Context, s *InterviewSession, expectedRevision int64) error
	CreateCalendarEvent(ctx context.Context, e *CalendarEvent) error
	UpdateCalendarEvent(ctx context.Context, e *CalendarEvent) error
	CountByApplication(ctx context.Context, userID, applicationID uuid.UUID) (int, error)
}

type CalendarEventFilter struct {
	ApplicationID *uuid.UUID
	From          *time.Time
	To            *time.Time
	Source        *EventSource
	Page          PageRequest
}

type CalendarEventStore interface {
	CreateCalendarEvent(ctx context.Context, e *CalendarEvent) error
	GetCalendarEvent(ctx context.Context, userID, id uuid.UUID) (*CalendarEvent, error)
	ListCalendarEvents(ctx context.Context, userID uuid.UUID, f CalendarEventFilter, limit int) ([]CalendarEvent, error)
	PatchCalendarEvent(ctx context.Context, e *CalendarEvent, expectedRevision int64) error
	DeleteCalendarEvent(ctx context.Context, userID, id uuid.UUID, expectedRevision int64) error
	UpsertExternalCalendarEvent(ctx context.Context, e *CalendarEvent) error
	DeleteExternalCalendarEvent(ctx context.Context, userID uuid.UUID, externalID string) error
}

type ApprovalFilter struct {
	ApplicationID *uuid.UUID
	Status        *ApprovalStatus
	Page          PageRequest
}

type ApprovalStore interface {
	Create(ctx context.Context, a *Approval) error
	Get(ctx context.Context, userID, id uuid.UUID) (*Approval, error)
	List(ctx context.Context, userID uuid.UUID, f ApprovalFilter, limit int) ([]Approval, error)
	Update(ctx context.Context, a *Approval, expectedRevision int64) error
	FindActive(ctx context.Context, userID uuid.UUID, kind ApprovalKind, applicationID, targetID uuid.UUID, now time.Time) (*Approval, error)
}

type ConversationFilter struct {
	ApplicationID *uuid.UUID
	Page          PageRequest
}

type ConversationStore interface {
	Create(ctx context.Context, c *Conversation) error
	Get(ctx context.Context, userID, id uuid.UUID) (*Conversation, error)
	List(ctx context.Context, userID uuid.UUID, f ConversationFilter, limit int) ([]Conversation, error)
	Update(ctx context.Context, c *Conversation, expectedRevision int64) error
	CreateMessage(ctx context.Context, m *Message) error
	ListMessages(ctx context.Context, userID, conversationID uuid.UUID, page PageRequest, limit int) ([]Message, error)
}

type OperationStore interface {
	Create(ctx context.Context, o *Operation) error
	Get(ctx context.Context, userID, id uuid.UUID) (*Operation, error)
	Update(ctx context.Context, o *Operation) error
}

type Source struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"-"`
	FileName  string    `json:"fileName"`
	MimeType  string    `json:"mimeType"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	Path      string    `json:"-"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type SourceStore interface {
	Create(ctx context.Context, s *Source) error
	Get(ctx context.Context, userID, id uuid.UUID) (*Source, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	Referenced(ctx context.Context, userID, id uuid.UUID) (bool, error)
}

type AiKey struct {
	UserID     uuid.UUID
	Provider   string
	LastFour   string
	Ciphertext []byte
	Nonce      []byte
	UpdatedAt  time.Time
}

type AiKeyStore interface {
	Put(ctx context.Context, k *AiKey) error
	Get(ctx context.Context, userID uuid.UUID, provider string) (*AiKey, error)
	List(ctx context.Context, userID uuid.UUID) ([]AiKey, error)
	Delete(ctx context.Context, userID uuid.UUID, provider string) error
}

type TokenCipher interface {
	Encrypt(plaintext string, userID uuid.UUID, purpose string) (ciphertext, nonce []byte, err error)
	Decrypt(ciphertext, nonce []byte, userID uuid.UUID, purpose string) (string, error)
}

type GoogleStore interface {
	SaveIntegrationCode(ctx context.Context, c *IntegrationCode) error
	GetIntegrationCode(ctx context.Context, code string) (*IntegrationCode, error)
	MarkIntegrationCodeUsed(ctx context.Context, code string, at time.Time) error
	PutGoogleIntegration(ctx context.Context, g *GoogleIntegration) error
	GetGoogleIntegration(ctx context.Context, userID uuid.UUID) (*GoogleIntegration, error)
	DeleteGoogleIntegration(ctx context.Context, userID uuid.UUID) error
	UpsertGoogleMessage(ctx context.Context, m *GoogleMessage) error
	GetGoogleMessage(ctx context.Context, userID, id uuid.UUID) (*GoogleMessage, error)
	ListGoogleMessages(ctx context.Context, userID uuid.UUID, applicationID *uuid.UUID, page PageRequest, limit int) ([]GoogleMessage, error)
	LinkGoogleMessage(ctx context.Context, userID, id, applicationID uuid.UUID) error
	DeleteGoogleData(ctx context.Context, userID uuid.UUID) error
}

type GoogleClient interface {
	ExchangeCode(ctx context.Context, code, redirectURI string) (*GoogleTokens, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (*GoogleTokens, error)
	RevokeToken(ctx context.Context, token string) error
	ListMessageIDs(ctx context.Context, accessToken, pageToken string) (ids []string, next string, historyID string, err error)
	GetMessage(ctx context.Context, accessToken, id string) (*GoogleMessageMeta, error)
	ListEvents(ctx context.Context, accessToken, syncToken string) (events []GoogleEvent, nextSyncToken string, err error)
}

type StripeCheckout struct {
	URL       string
	ExpiresAt *time.Time
}

type StripeGateway interface {
	CreateCheckoutSession(ctx context.Context, priceID, customerID, userID, planID, successURL, cancelURL string) (*StripeCheckout, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)
}

type BillingStore interface {
	RecordStripeEvent(ctx context.Context, eventID, typ string, at time.Time) (bool, error)
	UpdateSubscription(ctx context.Context, userID uuid.UUID, plan, status string, customerID *string, periodEndsAt *time.Time) error
	UpdateSubscriptionByCustomer(ctx context.Context, customerID, status string, periodEndsAt *time.Time) error
	UserIDByStripeCustomer(ctx context.Context, customerID string) (uuid.UUID, error)
	AppendLedger(ctx context.Context, e *LedgerEntry) error
	ListLedger(ctx context.Context, userID uuid.UUID, page PageRequest, limit int) ([]LedgerEntry, error)
	LastBalance(ctx context.Context, userID uuid.UUID) (int64, error)
}

type AIRequirement struct {
	Options *AiOptions
}

type AiOptions struct {
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	CredentialMode string `json:"credentialMode"`
	Effort         string `json:"effort"`
}

func ValidAiProvider(p string) bool {
	switch p {
	case "OPENAI", "CLAUDE", "GEMINI", "GROK":
		return true
	}
	return false
}

func ValidCredentialMode(m string) bool {
	return m == "MANAGED" || m == "BYOK"
}

func ValidEffort(e string) bool {
	return e == "LOW" || e == "MEDIUM" || e == "HIGH"
}

func (a *AiOptions) Validate() *Error {
	if a == nil {
		return nil
	}
	if !ValidAiProvider(a.Provider) {
		return ValidationField("ai.provider", "unsupported provider")
	}
	if a.Model == "" {
		return ValidationField("ai.model", "model is required")
	}
	if !ValidCredentialMode(a.CredentialMode) {
		return ValidationField("ai.credentialMode", "unsupported credentialMode")
	}
	if a.CredentialMode == "MANAGED" && a.Provider != "OPENAI" {
		return ValidationField("ai.credentialMode", "MANAGED is only supported for OPENAI")
	}
	if !ValidEffort(a.Effort) {
		return ValidationField("ai.effort", "unsupported effort")
	}
	return nil
}

func CheckAICredentials(ai *AiOptions, managedConfigured bool, byokConfigured func(provider string) bool) *Error {
	if ai == nil {
		return nil
	}
	if err := ai.Validate(); err != nil {
		return err
	}
	if ai.CredentialMode == "MANAGED" {
		if !managedConfigured {
			return NotConfigured("managed AI is not configured")
		}
		return nil
	}
	if !byokConfigured(ai.Provider) {
		return IntegrationRequired("no BYOK key configured for " + ai.Provider)
	}
	return nil
}
