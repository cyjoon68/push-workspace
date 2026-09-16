package domain

import (
	"time"

	"github.com/google/uuid"
)

type ApprovalKind string

const (
	ApprovalEvidenceUse       ApprovalKind = "EVIDENCE_USE"
	ApprovalDocumentFinalize  ApprovalKind = "DOCUMENT_FINALIZE"
	ApprovalApplicationSubmit ApprovalKind = "APPLICATION_SUBMIT"
	ApprovalCliExecute        ApprovalKind = "CLI_EXECUTE"
)

func ValidApprovalKind(k ApprovalKind) bool {
	switch k {
	case ApprovalEvidenceUse, ApprovalDocumentFinalize, ApprovalApplicationSubmit, ApprovalCliExecute:
		return true
	}
	return false
}

func OneShotApproval(k ApprovalKind) bool {
	return k != ApprovalEvidenceUse
}

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "PENDING"
	ApprovalApproved ApprovalStatus = "APPROVED"
	ApprovalDenied   ApprovalStatus = "DENIED"
	ApprovalExpired  ApprovalStatus = "EXPIRED"
	ApprovalConsumed ApprovalStatus = "CONSUMED"
)

type Approval struct {
	ID             uuid.UUID      `json:"id"`
	UserID         uuid.UUID      `json:"-"`
	Revision       int64          `json:"revision"`
	Kind           ApprovalKind   `json:"kind"`
	ApplicationID  uuid.UUID      `json:"applicationId"`
	TargetID       uuid.UUID      `json:"targetId"`
	TargetRevision *int64         `json:"targetRevision"`
	PayloadHash    string         `json:"payloadHash"`
	Status         ApprovalStatus `json:"status"`
	ExpiresAt      *time.Time     `json:"expiresAt"`
	DecidedAt      *time.Time     `json:"decidedAt"`
	ConsumedAt     *time.Time     `json:"consumedAt"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

const DefaultApprovalTTL = 24 * time.Hour

func (a *Approval) Expired(now time.Time) bool {
	return a.ExpiresAt != nil && now.After(*a.ExpiresAt)
}
