package domain

import (
	"time"

	"github.com/google/uuid"
)

type EvidenceKind string

const (
	EvidenceResume    EvidenceKind = "RESUME"
	EvidenceGithub    EvidenceKind = "GITHUB"
	EvidenceCareer    EvidenceKind = "CAREER"
	EvidenceEducation EvidenceKind = "EDUCATION"
	EvidenceSkill     EvidenceKind = "SKILL"
	EvidenceProject   EvidenceKind = "PROJECT"
)

func ValidEvidenceKind(k EvidenceKind) bool {
	switch k {
	case EvidenceResume, EvidenceGithub, EvidenceCareer, EvidenceEducation, EvidenceSkill, EvidenceProject:
		return true
	}
	return false
}

type VerificationStatus string

const (
	VerificationUserProvided VerificationStatus = "USER_PROVIDED"
	VerificationPending      VerificationStatus = "PENDING"
	VerificationVerified     VerificationStatus = "VERIFIED"
	VerificationRejected     VerificationStatus = "REJECTED"
)

type SourceLocation struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Unit  string `json:"unit"`
}

type Provenance struct {
	SourceID          *uuid.UUID      `json:"sourceId"`
	ProjectEvidenceID *uuid.UUID      `json:"projectEvidenceId"`
	ContentHash       string          `json:"contentHash"`
	SourceLocation    *SourceLocation `json:"sourceLocation"`
}

type CareerEvidence struct {
	ID                 uuid.UUID          `json:"id"`
	UserID             uuid.UUID          `json:"-"`
	Revision           int64              `json:"revision"`
	Kind               EvidenceKind       `json:"kind"`
	Title              string             `json:"title"`
	SourceText         string             `json:"sourceText"`
	SourceURL          *string            `json:"sourceUrl"`
	Skills             []string           `json:"skills"`
	VerificationStatus VerificationStatus `json:"verificationStatus"`
	Provenance         Provenance         `json:"provenance"`
	SupersedesID       *uuid.UUID         `json:"supersedesId"`
	Archived           bool               `json:"archived"`
	CreatedAt          time.Time          `json:"createdAt"`
	UpdatedAt          time.Time          `json:"updatedAt"`
}
