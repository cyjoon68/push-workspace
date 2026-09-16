package domain

import (
	"time"

	"github.com/google/uuid"
)

type CompanySource struct {
	SourceURL  string    `json:"sourceUrl"`
	SourceText string    `json:"sourceText"`
	AccessedAt time.Time `json:"accessedAt"`
}

type InterviewSession struct {
	ID              uuid.UUID       `json:"id"`
	UserID          uuid.UUID       `json:"-"`
	Revision        int64           `json:"revision"`
	ApplicationID   uuid.UUID       `json:"applicationId"`
	Title           string          `json:"title"`
	ScheduledAt     time.Time       `json:"scheduledAt"`
	DurationMinutes *int            `json:"durationMinutes"`
	EventID         *uuid.UUID      `json:"eventId"`
	EvidenceIDs     []uuid.UUID     `json:"evidenceIds"`
	CompanySources  []CompanySource `json:"companySources"`
	Notes           string          `json:"notes"`
	Reflection      string          `json:"reflection"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type CalendarEventType string

const (
	CalInterview CalendarEventType = "INTERVIEW"
	CalDeadline  CalendarEventType = "DEADLINE"
	CalFollowUp  CalendarEventType = "FOLLOW_UP"
	CalCustom    CalendarEventType = "CUSTOM"
)

type EventSource string

const (
	EventSourceLocal  EventSource = "LOCAL"
	EventSourceGoogle EventSource = "GOOGLE"
)

type CalendarEvent struct {
	ID            uuid.UUID         `json:"id"`
	UserID        uuid.UUID         `json:"-"`
	Revision      int64             `json:"revision"`
	ApplicationID *uuid.UUID        `json:"applicationId"`
	Type          CalendarEventType `json:"type"`
	Title         string            `json:"title"`
	StartsAt      time.Time         `json:"startsAt"`
	EndsAt        time.Time         `json:"endsAt"`
	TimeZone      string            `json:"timeZone"`
	Source        EventSource       `json:"source"`
	ExternalID    *string           `json:"externalId"`
	Notes         string            `json:"notes"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}
