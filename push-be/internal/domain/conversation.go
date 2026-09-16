package domain

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"-"`
	Revision      int64      `json:"revision"`
	ApplicationID *uuid.UUID `json:"applicationId"`
	Title         string     `json:"title"`
	Pinned        bool       `json:"pinned"`
	Archived      bool       `json:"-"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type MessageRole string

const (
	RoleUser      MessageRole = "USER"
	RoleAssistant MessageRole = "ASSISTANT"
	RoleSystem    MessageRole = "SYSTEM"
)

type AttachmentType string

const (
	AttachDocumentVersion AttachmentType = "DOCUMENT_VERSION"
	AttachEvidence        AttachmentType = "EVIDENCE"
	AttachApproval        AttachmentType = "APPROVAL"
)

type MessageAttachment struct {
	Type       AttachmentType `json:"type"`
	ID         uuid.UUID      `json:"id"`
	DocumentID *uuid.UUID     `json:"documentId,omitempty"`
	Title      string         `json:"title,omitempty"`
}

type Message struct {
	ID             uuid.UUID           `json:"id"`
	UserID         uuid.UUID           `json:"-"`
	ConversationID uuid.UUID           `json:"conversationId"`
	Role           MessageRole         `json:"role"`
	Text           string              `json:"text"`
	Attachments    []MessageAttachment `json:"attachments"`
	OperationID    *uuid.UUID          `json:"operationId"`
	CreatedAt      time.Time           `json:"createdAt"`
}

type AccessMode string

const (
	AccessSuggest AccessMode = "SUGGEST"
	AccessConfirm AccessMode = "CONFIRM_ACTIONS"
)

func ValidAccessMode(m AccessMode) bool {
	return m == AccessSuggest || m == AccessConfirm
}
