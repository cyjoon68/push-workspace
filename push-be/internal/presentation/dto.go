package presentation

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type exchangeReq struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"codeVerifier"`
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

type logoutReq struct {
	RefreshToken string `json:"refreshToken"`
}

type createEvidenceReq struct {
	Kind         string     `json:"kind"`
	Title        string     `json:"title"`
	SourceText   string     `json:"sourceText"`
	SourceURL    *string    `json:"sourceUrl"`
	Skills       []string   `json:"skills"`
	SupersedesID *uuid.UUID `json:"supersedesId"`
}

type importEvidenceReq struct {
	SourceID    *uuid.UUID `json:"sourceId"`
	Text        string     `json:"text"`
	SourceURL   *string    `json:"sourceUrl"`
	ContentHash string     `json:"contentHash"`
	Format      string     `json:"format"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
}

type expectedRevisionReq struct {
	ExpectedRevision int64 `json:"expectedRevision"`
}

type createJobReq struct {
	Company      string          `json:"company"`
	Title        string          `json:"title"`
	SourceKind   string          `json:"sourceKind"`
	SourceURL    *string         `json:"sourceUrl"`
	SourceText   string          `json:"sourceText"`
	Requirements []string        `json:"requirements"`
	Preferred    []string        `json:"preferred"`
	Deadline     json.RawMessage `json:"deadline"`
	Language     string          `json:"language"`
}

type patchJobReq struct {
	ExpectedRevision int64           `json:"expectedRevision"`
	Company          *string         `json:"company"`
	Title            *string         `json:"title"`
	Requirements     *[]string       `json:"requirements"`
	Preferred        *[]string       `json:"preferred"`
	Deadline         json.RawMessage `json:"deadline"`
}

type analyzeReq struct {
	ApplicationID    uuid.UUID         `json:"applicationId"`
	ExpectedRevision int64             `json:"expectedRevision"`
	EvidenceIDs      []uuid.UUID       `json:"evidenceIds"`
	AI               *domain.AiOptions `json:"ai"`
}

type createApplicationReq struct {
	JobID uuid.UUID `json:"jobId"`
	Notes string    `json:"notes"`
}

type importApplicationReq struct {
	JobID     uuid.UUID `json:"jobId"`
	Stage     string    `json:"stage"`
	AppliedAt time.Time `json:"appliedAt"`
	Notes     string    `json:"notes"`
	Confirmed bool      `json:"confirmed"`
}

type patchApplicationReq struct {
	ExpectedRevision int64           `json:"expectedRevision"`
	Stage            *string         `json:"stage"`
	Notes            *string         `json:"notes"`
	NextActionAt     json.RawMessage `json:"nextActionAt"`
}

type createDraftReq struct {
	ExpectedRevision   int64       `json:"expectedRevision"`
	Mode               string      `json:"mode"`
	Adapter            *string     `json:"adapter"`
	DocumentVersionIDs []uuid.UUID `json:"documentVersionIds"`
	ConfirmedSubmitted bool        `json:"confirmedSubmitted"`
}

type submitReq struct {
	ExpectedRevision int64     `json:"expectedRevision"`
	DraftID          uuid.UUID `json:"draftId"`
	ApprovalID       uuid.UUID `json:"approvalId"`
}

type createDocumentReq struct {
	ApplicationID uuid.UUID `json:"applicationId"`
	Title         string    `json:"title"`
	Kind          string    `json:"kind"`
	Template      string    `json:"template"`
	Language      string    `json:"language"`
}

type patchDocumentReq struct {
	ExpectedRevision int64   `json:"expectedRevision"`
	Title            *string `json:"title"`
	Template         *string `json:"template"`
	Language         *string `json:"language"`
}

type blockInput struct {
	ID           string               `json:"id"`
	Text         string               `json:"text"`
	EvidenceRefs []domain.EvidenceRef `json:"evidenceRefs"`
}

type createVersionReq struct {
	ExpectedRevision int64           `json:"expectedRevision"`
	Content          json.RawMessage `json:"content"`
	Blocks           []blockInput    `json:"blocks"`
	ChangeNote       string          `json:"changeNote"`
}

type generateDocReq struct {
	ExpectedRevision int64             `json:"expectedRevision"`
	EvidenceIDs      []uuid.UUID       `json:"evidenceIds"`
	AnalysisID       *uuid.UUID        `json:"analysisId"`
	AI               *domain.AiOptions `json:"ai"`
	Language         *string           `json:"language"`
}

type reviseReq struct {
	ExpectedRevision int64             `json:"expectedRevision"`
	VersionID        uuid.UUID         `json:"versionId"`
	Selection        domain.Selection  `json:"selection"`
	Action           string            `json:"action"`
	Instruction      string            `json:"instruction"`
	AI               *domain.AiOptions `json:"ai"`
}

type reviewReq struct {
	VersionID uuid.UUID `json:"versionId"`
}

type finalizeReq struct {
	ExpectedRevision int64     `json:"expectedRevision"`
	VersionID        uuid.UUID `json:"versionId"`
	ApprovalID       uuid.UUID `json:"approvalId"`
}

type createExportReq struct {
	VersionID       uuid.UUID `json:"versionId"`
	Format          string    `json:"format"`
	RendererVersion string    `json:"rendererVersion"`
}

type exportResultReq struct {
	SHA256     string `json:"sha256"`
	ByteLength int64  `json:"byteLength"`
	PageCount  *int   `json:"pageCount"`
	Validation struct {
		KoreanText bool `json:"koreanText"`
		Links      bool `json:"links"`
		AtsText    bool `json:"atsText"`
	} `json:"validation"`
	Status    string  `json:"status"`
	ErrorCode *string `json:"errorCode"`
}

type blueprintsReq struct {
	ApplicationID uuid.UUID         `json:"applicationId"`
	GapAnalysisID uuid.UUID         `json:"gapAnalysisId"`
	AI            *domain.AiOptions `json:"ai"`
}

type createRunReq struct {
	Provider         string `json:"provider"`
	WorkingDirectory string `json:"workingDirectory"`
	Prompt           string `json:"prompt"`
}

type startRunReq struct {
	ExpectedRevision int64     `json:"expectedRevision"`
	ApprovalID       uuid.UUID `json:"approvalId"`
	DetectedVersion  string    `json:"detectedVersion"`
	DeviceID         string    `json:"deviceId"`
}

type launchReq struct {
	ExpectedRevision int64               `json:"expectedRevision"`
	DeviceID         string              `json:"deviceId"`
	PayloadHash      string              `json:"payloadHash"`
	LaunchStatus     string              `json:"launchStatus"`
	Process          *domain.ProcessInfo `json:"process"`
}

type recoverReq struct {
	ExpectedRevision int64               `json:"expectedRevision"`
	DeviceID         string              `json:"deviceId"`
	Decision         string              `json:"decision"`
	Process          *domain.ProcessInfo `json:"process"`
	FailureReason    *string             `json:"failureReason"`
}

type runResultReq struct {
	ExpectedRevision int64   `json:"expectedRevision"`
	ExitCode         int     `json:"exitCode"`
	CommitSHA        *string `json:"commitSha"`
	StdoutHash       string  `json:"stdoutHash"`
	StderrHash       string  `json:"stderrHash"`
}

type createProjectEvidenceReq struct {
	RunID       uuid.UUID            `json:"runId"`
	CommitURL   string               `json:"commitUrl"`
	TestCommand string               `json:"testCommand"`
	TestOutput  string               `json:"testOutput"`
	ExitCode    int                  `json:"exitCode"`
	Metrics     []domain.MetricEntry `json:"metrics"`
	Summary     string               `json:"summary"`
}

type createInterviewReq struct {
	ApplicationID   uuid.UUID              `json:"applicationId"`
	Title           string                 `json:"title"`
	ScheduledAt     time.Time              `json:"scheduledAt"`
	DurationMinutes *int                   `json:"durationMinutes"`
	EvidenceIDs     []uuid.UUID            `json:"evidenceIds"`
	CompanySources  []domain.CompanySource `json:"companySources"`
	Notes           string                 `json:"notes"`
	TimeZone        string                 `json:"timeZone"`
}

type patchInterviewReq struct {
	ExpectedRevision int64                   `json:"expectedRevision"`
	Title            *string                 `json:"title"`
	ScheduledAt      *time.Time              `json:"scheduledAt"`
	CompanySources   *[]domain.CompanySource `json:"companySources"`
	Notes            *string                 `json:"notes"`
	Reflection       *string                 `json:"reflection"`
}

type prepareReq struct {
	ExpectedRevision int64             `json:"expectedRevision"`
	AI               *domain.AiOptions `json:"ai"`
}

type createApprovalReq struct {
	Kind           string    `json:"kind"`
	ApplicationID  uuid.UUID `json:"applicationId"`
	TargetID       uuid.UUID `json:"targetId"`
	TargetRevision *int64    `json:"targetRevision"`
}

type decisionReq struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	Decision         string `json:"decision"`
}

type opInputReq struct {
	Fields   map[string]string `json:"fields"`
	SourceID *uuid.UUID        `json:"sourceId"`
}

type putAiKeyReq struct {
	Key string `json:"key"`
}

type aiGenerateReq struct {
	AI            *domain.AiOptions `json:"ai"`
	Prompt        string            `json:"prompt"`
	ApplicationID uuid.UUID         `json:"applicationId"`
	EvidenceIDs   []uuid.UUID       `json:"evidenceIds"`
}

type googleConnectReq struct {
	CodeChallenge       string `json:"codeChallenge"`
	CodeChallengeMethod string `json:"codeChallengeMethod"`
	RedirectURI         string `json:"redirectUri"`
}

type googleCompleteReq struct {
	IntegrationCode string `json:"integrationCode"`
	CodeVerifier    string `json:"codeVerifier"`
}

type linkMessageReq struct {
	ApplicationID uuid.UUID `json:"applicationId"`
}

type checkoutReq struct {
	PlanID string `json:"planId"`
}

type createCalendarEventReq struct {
	ApplicationID *uuid.UUID `json:"applicationId"`
	Type          string     `json:"type"`
	Title         string     `json:"title"`
	StartsAt      time.Time  `json:"startsAt"`
	EndsAt        time.Time  `json:"endsAt"`
	TimeZone      string     `json:"timeZone"`
	Notes         string     `json:"notes"`
}

type patchCalendarEventReq struct {
	ExpectedRevision int64      `json:"expectedRevision"`
	Title            *string    `json:"title"`
	StartsAt         *time.Time `json:"startsAt"`
	EndsAt           *time.Time `json:"endsAt"`
	TimeZone         *string    `json:"timeZone"`
	Notes            *string    `json:"notes"`
}

type createConversationReq struct {
	ApplicationID *uuid.UUID `json:"applicationId"`
	Title         string     `json:"title"`
}

type patchConversationReq struct {
	ExpectedRevision int64   `json:"expectedRevision"`
	Title            *string `json:"title"`
	Pinned           *bool   `json:"pinned"`
}

type messageContextReq struct {
	DocumentID  *uuid.UUID  `json:"documentId"`
	VersionID   *uuid.UUID  `json:"versionId"`
	EvidenceIDs []uuid.UUID `json:"evidenceIds"`
}

type postMessageReq struct {
	Text       string             `json:"text"`
	Context    *messageContextReq `json:"context"`
	AI         *domain.AiOptions  `json:"ai"`
	AccessMode string             `json:"accessMode"`
}
