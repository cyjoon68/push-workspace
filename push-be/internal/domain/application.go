package domain

import (
	"time"

	"github.com/google/uuid"
)

type ApplicationStage string

const (
	StageDiscovered ApplicationStage = "DISCOVERED"
	StagePreparing  ApplicationStage = "PREPARING"
	StageReady      ApplicationStage = "READY"
	StageApplied    ApplicationStage = "APPLIED"
	StageScreening  ApplicationStage = "SCREENING"
	StageInterview  ApplicationStage = "INTERVIEW"
	StageOffer      ApplicationStage = "OFFER"
	StageAccepted   ApplicationStage = "ACCEPTED"
	StageRejected   ApplicationStage = "REJECTED"
	StageWithdrawn  ApplicationStage = "WITHDRAWN"
)

var stageOrder = []ApplicationStage{
	StageDiscovered, StagePreparing, StageReady, StageApplied,
	StageScreening, StageInterview, StageOffer, StageAccepted,
}

func stageIndex(s ApplicationStage) int {
	for i, v := range stageOrder {
		if v == s {
			return i
		}
	}
	return -1
}

func ValidStage(s ApplicationStage) bool {
	return stageIndex(s) >= 0 || s == StageRejected || s == StageWithdrawn
}

func TerminalStage(s ApplicationStage) bool {
	return s == StageAccepted || s == StageRejected || s == StageWithdrawn
}

func CanPatchStage(from, to ApplicationStage) bool {
	if from == to {
		return true
	}
	if TerminalStage(from) {
		return false
	}
	if to == StageRejected || to == StageWithdrawn {
		return true
	}
	if to == StageApplied {
		return false
	}
	return stageIndex(to) == stageIndex(from)+1
}

func NextStage(s ApplicationStage) (ApplicationStage, bool) {
	i := stageIndex(s)
	if i < 0 || i+1 >= len(stageOrder) {
		return "", false
	}
	return stageOrder[i+1], true
}

type Application struct {
	ID           uuid.UUID        `json:"id"`
	UserID       uuid.UUID        `json:"-"`
	Revision     int64            `json:"revision"`
	JobID        uuid.UUID        `json:"jobId"`
	Company      string           `json:"company"`
	Title        string           `json:"title"`
	Stage        ApplicationStage `json:"stage"`
	Notes        string           `json:"notes"`
	AppliedAt    *time.Time       `json:"appliedAt"`
	NextActionAt *time.Time       `json:"nextActionAt"`
	Imported     bool             `json:"imported"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

type ApplicationEvent struct {
	ID            uuid.UUID      `json:"id"`
	UserID        uuid.UUID      `json:"-"`
	ApplicationID uuid.UUID      `json:"applicationId"`
	Type          string         `json:"type"`
	Payload       map[string]any `json:"payload"`
	CreatedAt     time.Time      `json:"createdAt"`
}

const (
	EventStageChanged  = "STAGE_CHANGED"
	EventImported      = "IMPORTED"
	EventSubmission    = "SUBMISSION"
	EventDocumentFinal = "DOCUMENT_FINALIZED"
	EventApproval      = "APPROVAL"
	EventInterview     = "INTERVIEW"
)

type SubmissionMode string

const (
	SubmissionManual  SubmissionMode = "MANUAL_RECORD"
	SubmissionAdapter SubmissionMode = "ADAPTER"
)

type SubmissionDraft struct {
	ID                  uuid.UUID      `json:"id"`
	UserID              uuid.UUID      `json:"-"`
	ApplicationID       uuid.UUID      `json:"applicationId"`
	ApplicationRevision int64          `json:"applicationRevision"`
	Mode                SubmissionMode `json:"mode"`
	Adapter             *string        `json:"adapter"`
	DocumentVersionIDs  []uuid.UUID    `json:"documentVersionIds"`
	ConfirmedSubmitted  bool           `json:"confirmedSubmitted"`
	PayloadHash         string         `json:"payloadHash"`
	CreatedAt           time.Time      `json:"createdAt"`
}

type SubmissionStatus string

const (
	SubmissionPending   SubmissionStatus = "PENDING"
	SubmissionRunning   SubmissionStatus = "RUNNING"
	SubmissionSucceeded SubmissionStatus = "SUCCEEDED"
	SubmissionFailed    SubmissionStatus = "FAILED"
	SubmissionUnknown   SubmissionStatus = "UNKNOWN"
)

type Submission struct {
	ID                 uuid.UUID        `json:"id"`
	UserID             uuid.UUID        `json:"-"`
	ApplicationID      uuid.UUID        `json:"applicationId"`
	Mode               SubmissionMode   `json:"mode"`
	Adapter            *string          `json:"adapter"`
	Status             SubmissionStatus `json:"status"`
	DocumentVersionIDs []uuid.UUID      `json:"documentVersionIds"`
	ApprovalID         uuid.UUID        `json:"approvalId"`
	ReceiptURL         *string          `json:"receiptUrl"`
	ErrorCode          *string          `json:"errorCode"`
	CreatedAt          time.Time        `json:"createdAt"`
	UpdatedAt          time.Time        `json:"updatedAt"`
}
