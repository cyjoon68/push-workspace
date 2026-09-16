package domain

import (
	"time"

	"github.com/google/uuid"
)

type JobSourceKind string

const (
	JobSourceURL  JobSourceKind = "URL"
	JobSourceText JobSourceKind = "TEXT"
	JobSourceDOM  JobSourceKind = "DOM"
)

func ValidJobSourceKind(k JobSourceKind) bool {
	return k == JobSourceURL || k == JobSourceText || k == JobSourceDOM
}

type JobPosting struct {
	ID           uuid.UUID     `json:"id"`
	UserID       uuid.UUID     `json:"-"`
	Revision     int64         `json:"revision"`
	Company      string        `json:"company"`
	Title        string        `json:"title"`
	SourceKind   JobSourceKind `json:"sourceKind"`
	SourceURL    *string       `json:"sourceUrl"`
	SourceText   string        `json:"sourceText"`
	Requirements []string      `json:"requirements"`
	Preferred    []string      `json:"preferred"`
	Keywords     []string      `json:"keywords"`
	Risks        []string      `json:"risks"`
	Deadline     *time.Time    `json:"deadline"`
	Language     string        `json:"language"`
	Archived     bool          `json:"archived"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

type AnalysisMethod string

const (
	MethodRuleBased  AnalysisMethod = "RULE_BASED"
	MethodAIAssisted AnalysisMethod = "AI_ASSISTED"
)

type RequirementMatch struct {
	Requirement string   `json:"requirement"`
	EvidenceIDs []string `json:"evidenceIds"`
}

type GapAnalysis struct {
	ID               uuid.UUID          `json:"id"`
	UserID           uuid.UUID          `json:"-"`
	ApplicationID    uuid.UUID          `json:"applicationId"`
	JobID            uuid.UUID          `json:"jobId"`
	JobRevision      int64              `json:"jobRevision"`
	EvidenceIDs      []uuid.UUID        `json:"evidenceIds"`
	Matched          []RequirementMatch `json:"matched"`
	Missing          []string           `json:"missing"`
	PreferredMissing []string           `json:"preferredMissing"`
	Risks            []string           `json:"risks"`
	FitScore         *int               `json:"fitScore"`
	Method           AnalysisMethod     `json:"method"`
	Stale            bool               `json:"stale"`
	CreatedAt        time.Time          `json:"createdAt"`
}
