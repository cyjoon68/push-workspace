package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OperationType string

const (
	OpEvidenceImport    OperationType = "EVIDENCE_IMPORT"
	OpJobAnalysis       OperationType = "JOB_ANALYSIS"
	OpDocumentGenerate  OperationType = "DOCUMENT_GENERATE"
	OpDocumentRevise    OperationType = "DOCUMENT_REVISE"
	OpProjectBlueprints OperationType = "PROJECT_BLUEPRINTS"
	OpProjectVerify     OperationType = "PROJECT_VERIFY"
	OpInterviewPrepare  OperationType = "INTERVIEW_PREPARE"
	OpAIGenerate        OperationType = "AI_GENERATE"
	OpChatMessage       OperationType = "CHAT_MESSAGE"
	OpGoogleSync        OperationType = "GOOGLE_SYNC"
	OpApplicationSubmit OperationType = "APPLICATION_SUBMIT"
)

type OperationStatus string

const (
	OpQueued     OperationStatus = "QUEUED"
	OpRunning    OperationStatus = "RUNNING"
	OpNeedsInput OperationStatus = "NEEDS_INPUT"
	OpSucceeded  OperationStatus = "SUCCEEDED"
	OpFailed     OperationStatus = "FAILED"
	OpCancelled  OperationStatus = "CANCELLED"
)

func TerminalOperationStatus(s OperationStatus) bool {
	return s == OpSucceeded || s == OpFailed || s == OpCancelled
}

type InputField struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type InputRequest struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []InputField `json:"fields"`
}

type OperationError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type OperationResult struct {
	Kind  string          `json:"kind"`
	Value json.RawMessage `json:"value"`
}

type Operation struct {
	ID             uuid.UUID        `json:"id"`
	UserID         uuid.UUID        `json:"-"`
	Type           OperationType    `json:"type"`
	ApplicationID  *uuid.UUID       `json:"applicationId"`
	Status         OperationStatus  `json:"status"`
	Progress       *int             `json:"progress"`
	Result         *OperationResult `json:"result"`
	Error          *OperationError  `json:"error"`
	InputRequest   *InputRequest    `json:"inputRequest"`
	PendingPayload json.RawMessage  `json:"-"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}
