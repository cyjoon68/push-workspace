package domain

import (
	"time"

	"github.com/google/uuid"
)

type CliProvider string

const (
	CliCodex      CliProvider = "CODEX"
	CliClaudeCode CliProvider = "CLAUDE_CODE"
	CliGrokBuild  CliProvider = "GROK_BUILD"
)

func ValidCliProvider(p CliProvider) bool {
	return p == CliCodex || p == CliClaudeCode || p == CliGrokBuild
}

type BlueprintState string

const (
	BlueprintDraft      BlueprintState = "DRAFT"
	BlueprintSelected   BlueprintState = "SELECTED"
	BlueprintInProgress BlueprintState = "IN_PROGRESS"
	BlueprintVerified   BlueprintState = "VERIFIED"
	BlueprintArchived   BlueprintState = "ARCHIVED"
)

type CliRunState string

const (
	CliRunDraft            CliRunState = "DRAFT"
	CliRunApprovalRequired CliRunState = "APPROVAL_REQUIRED"
	CliRunRunning          CliRunState = "RUNNING"
	CliRunVerifying        CliRunState = "VERIFYING"
	CliRunVerified         CliRunState = "VERIFIED"
	CliRunFailed           CliRunState = "FAILED"
)

type LaunchStatus string

const (
	LaunchNotClaimed LaunchStatus = "NOT_CLAIMED"
	LaunchClaimed    LaunchStatus = "CLAIMED"
	LaunchStarted    LaunchStatus = "STARTED"
	LaunchUnknown    LaunchStatus = "UNKNOWN"
	LaunchFinished   LaunchStatus = "FINISHED"
)

func ValidReportedLaunchStatus(s LaunchStatus) bool {
	return s == LaunchClaimed || s == LaunchStarted || s == LaunchUnknown
}

func CanTransitionLaunch(from, to LaunchStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case LaunchNotClaimed:
		return to == LaunchClaimed
	case LaunchClaimed:
		return to == LaunchStarted || to == LaunchUnknown
	case LaunchStarted:
		return to == LaunchUnknown
	}
	return false
}

type BlueprintTask struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Acceptance  []string `json:"acceptance"`
}

type BlueprintMetric struct {
	Name        string   `json:"name"`
	Unit        string   `json:"unit"`
	Measurement string   `json:"measurement"`
	Target      *float64 `json:"target"`
}

type EffortEstimate struct {
	MinHours int `json:"minHours"`
	MaxHours int `json:"maxHours"`
}

type ProjectBlueprint struct {
	ID                 uuid.UUID         `json:"id"`
	UserID             uuid.UUID         `json:"-"`
	Revision           int64             `json:"revision"`
	ApplicationID      uuid.UUID         `json:"applicationId"`
	GapAnalysisID      uuid.UUID         `json:"gapAnalysisId"`
	Title              string            `json:"title"`
	Skills             []string          `json:"skills"`
	Problem            string            `json:"problem"`
	Solution           string            `json:"solution"`
	Tasks              []BlueprintTask   `json:"tasks"`
	CompletionCriteria []string          `json:"completionCriteria"`
	Metrics            []BlueprintMetric `json:"metrics"`
	EstimatedEffort    EffortEstimate    `json:"estimatedEffort"`
	State              BlueprintState    `json:"state"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type ProcessInfo struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
}

type CliRun struct {
	ID               uuid.UUID    `json:"id"`
	UserID           uuid.UUID    `json:"-"`
	Revision         int64        `json:"revision"`
	ProjectID        uuid.UUID    `json:"projectId"`
	ApplicationID    uuid.UUID    `json:"applicationId"`
	Provider         CliProvider  `json:"provider"`
	WorkingDirectory string       `json:"workingDirectory"`
	Executable       string       `json:"executable"`
	Arguments        []string     `json:"arguments"`
	Prompt           string       `json:"prompt"`
	PayloadHash      string       `json:"payloadHash"`
	State            CliRunState  `json:"state"`
	LaunchStatus     LaunchStatus `json:"launchStatus"`
	ApprovalID       *uuid.UUID   `json:"approvalId"`
	DeviceID         *string      `json:"deviceId"`
	DetectedVersion  *string      `json:"detectedVersion"`
	StartedAt        *time.Time   `json:"startedAt"`
	FinishedAt       *time.Time   `json:"finishedAt"`
	FailureReason    *string      `json:"failureReason"`
	ExitCode         *int         `json:"exitCode"`
	CommitSHA        *string      `json:"commitSha"`
	StdoutHash       *string      `json:"stdoutHash"`
	StderrHash       *string      `json:"stderrHash"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

type MetricEntry struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type TestResults struct {
	TestCommand string `json:"testCommand"`
	TestOutput  string `json:"testOutput"`
	ExitCode    int    `json:"exitCode"`
}

type ProjectEvidenceStatus string

const (
	ProjectEvidencePending  ProjectEvidenceStatus = "PENDING"
	ProjectEvidenceVerified ProjectEvidenceStatus = "VERIFIED"
	ProjectEvidenceRejected ProjectEvidenceStatus = "REJECTED"
)

type ProjectEvidence struct {
	ID                 uuid.UUID             `json:"id"`
	UserID             uuid.UUID             `json:"-"`
	Revision           int64                 `json:"revision"`
	ProjectID          uuid.UUID             `json:"projectId"`
	RunID              uuid.UUID             `json:"runId"`
	CommitURL          string                `json:"commitUrl"`
	CommitSHA          *string               `json:"commitSha"`
	TestResults        TestResults           `json:"testResults"`
	Metrics            []MetricEntry         `json:"metrics"`
	Summary            string                `json:"summary"`
	Status             ProjectEvidenceStatus `json:"status"`
	VerificationMethod *string               `json:"verificationMethod"`
	VerifiedAt         *time.Time            `json:"verifiedAt"`
	CareerEvidenceID   *uuid.UUID            `json:"careerEvidenceId"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
}
