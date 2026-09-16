package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DocumentKind string

const (
	DocResume      DocumentKind = "RESUME"
	DocPortfolio   DocumentKind = "PORTFOLIO"
	DocCoverLetter DocumentKind = "COVER_LETTER"
)

func ValidDocumentKind(k DocumentKind) bool {
	return k == DocResume || k == DocPortfolio || k == DocCoverLetter
}

type DocumentTemplate string

const (
	TemplateClassic DocumentTemplate = "CLASSIC"
	TemplateModern  DocumentTemplate = "MODERN"
	TemplateCompact DocumentTemplate = "COMPACT"
)

func ValidDocumentTemplate(t DocumentTemplate) bool {
	return t == TemplateClassic || t == TemplateModern || t == TemplateCompact
}

type DocumentStatus string

const (
	DocStatusDraft     DocumentStatus = "DRAFT"
	DocStatusFinalized DocumentStatus = "FINALIZED"
	DocStatusArchived  DocumentStatus = "ARCHIVED"
)

type Document struct {
	ID                 uuid.UUID        `json:"id"`
	UserID             uuid.UUID        `json:"-"`
	Revision           int64            `json:"revision"`
	ApplicationID      uuid.UUID        `json:"applicationId"`
	Title              string           `json:"title"`
	Kind               DocumentKind     `json:"kind"`
	Template           DocumentTemplate `json:"template"`
	Language           string           `json:"language"`
	Status             DocumentStatus   `json:"status"`
	LatestVersionID    *uuid.UUID       `json:"latestVersionId"`
	FinalizedVersionID *uuid.UUID       `json:"finalizedVersionId"`
	CreatedAt          time.Time        `json:"createdAt"`
	UpdatedAt          time.Time        `json:"updatedAt"`
}

type ClaimStatus string

const (
	ClaimSupported   ClaimStatus = "SUPPORTED"
	ClaimNeedsReview ClaimStatus = "NEEDS_REVIEW"
	ClaimUnsupported ClaimStatus = "UNSUPPORTED"
)

type EvidenceRef struct {
	EvidenceID uuid.UUID `json:"evidenceId"`
	Start      int       `json:"start"`
	End        int       `json:"end"`
}

type Block struct {
	ID           string        `json:"id"`
	Text         string        `json:"text"`
	EvidenceRefs []EvidenceRef `json:"evidenceRefs"`
	ClaimStatus  ClaimStatus   `json:"claimStatus"`
}

type QualityIssue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	BlockID  string `json:"blockId,omitempty"`
	Message  string `json:"message"`
}

type Quality struct {
	JobFit           *int           `json:"jobFit"`
	EvidenceFidelity *int           `json:"evidenceFidelity"`
	Readability      *int           `json:"readability"`
	Ats              *int           `json:"ats"`
	Method           AnalysisMethod `json:"method"`
	Issues           []QualityIssue `json:"issues"`
}

type DocumentVersion struct {
	ID            uuid.UUID       `json:"id"`
	UserID        uuid.UUID       `json:"-"`
	DocumentID    uuid.UUID       `json:"documentId"`
	ApplicationID uuid.UUID       `json:"applicationId"`
	Number        int             `json:"number"`
	Content       json.RawMessage `json:"content"`
	Blocks        []Block         `json:"blocks"`
	ChangeNote    string          `json:"changeNote"`
	Quality       Quality         `json:"quality"`
	CreatedAt     time.Time       `json:"createdAt"`
}

type RevisionAction string

const (
	RevRewrite     RevisionAction = "REWRITE"
	RevShorten     RevisionAction = "SHORTEN"
	RevEmphasize   RevisionAction = "EMPHASIZE_METRICS"
	RevChangeTone  RevisionAction = "CHANGE_TONE"
	RevTailorToJob RevisionAction = "TAILOR_TO_JOB"
)

func ValidRevisionAction(a RevisionAction) bool {
	switch a {
	case RevRewrite, RevShorten, RevEmphasize, RevChangeTone, RevTailorToJob:
		return true
	}
	return false
}

type Selection struct {
	From int    `json:"from"`
	To   int    `json:"to"`
	Text string `json:"text"`
}

type RevisionProposal struct {
	ID              uuid.UUID     `json:"id"`
	UserID          uuid.UUID     `json:"-"`
	DocumentID      uuid.UUID     `json:"documentId"`
	SourceVersionID uuid.UUID     `json:"sourceVersionId"`
	SourceRevision  int64         `json:"sourceRevision"`
	Selection       Selection     `json:"selection"`
	Replacement     string        `json:"replacement"`
	EvidenceRefs    []EvidenceRef `json:"evidenceRefs"`
	ClaimStatus     ClaimStatus   `json:"claimStatus"`
	AppliedAt       *time.Time    `json:"appliedAt"`
	CreatedAt       time.Time     `json:"createdAt"`
}

type ExportFormat string

const (
	ExportPDF  ExportFormat = "PDF"
	ExportDOCX ExportFormat = "DOCX"
)

type ExportStatus string

const (
	ExportReadyToRender ExportStatus = "READY_TO_RENDER"
	ExportSucceeded     ExportStatus = "SUCCEEDED"
	ExportFailed        ExportStatus = "FAILED"
)

type DocumentExport struct {
	ID              uuid.UUID        `json:"id"`
	UserID          uuid.UUID        `json:"-"`
	DocumentID      uuid.UUID        `json:"documentId"`
	VersionID       uuid.UUID        `json:"versionId"`
	Format          ExportFormat     `json:"format"`
	Template        DocumentTemplate `json:"template"`
	Language        string           `json:"language"`
	Content         json.RawMessage  `json:"content"`
	Blocks          []Block          `json:"blocks"`
	ContentHash     string           `json:"contentHash"`
	Status          ExportStatus     `json:"status"`
	RendererVersion string           `json:"rendererVersion"`
	Result          *ExportResult    `json:"result"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

type ExportResult struct {
	SHA256     string           `json:"sha256"`
	ByteLength int64            `json:"byteLength"`
	PageCount  *int             `json:"pageCount,omitempty"`
	Validation ExportValidation `json:"validation"`
	ErrorCode  *string          `json:"errorCode,omitempty"`
}

type ExportValidation struct {
	KoreanText bool `json:"koreanText"`
	Links      bool `json:"links"`
	AtsText    bool `json:"atsText"`
}
