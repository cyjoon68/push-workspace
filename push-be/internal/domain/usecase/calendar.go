package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const maxCalendarRangeDays = 366

type CalendarService struct {
	events       domain.CalendarEventStore
	applications domain.ApplicationStore
}

func NewCalendarService(events domain.CalendarEventStore, applications domain.ApplicationStore) *CalendarService {
	return &CalendarService{events: events, applications: applications}
}

func validEventType(t domain.CalendarEventType) bool {
	switch t {
	case domain.CalInterview, domain.CalDeadline, domain.CalFollowUp, domain.CalCustom:
		return true
	}
	return false
}

func (s *CalendarService) List(ctx context.Context, userID uuid.UUID, f domain.CalendarEventFilter) (domain.Page[domain.CalendarEvent], error) {
	if f.From == nil || f.To == nil {
		return domain.Page[domain.CalendarEvent]{}, domain.Validation("from and to are required")
	}
	if !f.To.After(*f.From) {
		return domain.Page[domain.CalendarEvent]{}, domain.ValidationField("to", "must be after from")
	}
	if f.To.Sub(*f.From) > maxCalendarRangeDays*24*time.Hour {
		return domain.Page[domain.CalendarEvent]{}, domain.ValidationField("to", "range must be at most 366 days")
	}
	limit := f.Page.EffectiveLimit()
	items, err := s.events.ListCalendarEvents(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.CalendarEvent]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(e domain.CalendarEvent) domain.Cursor {
		return domain.Cursor{CreatedAt: e.CreatedAt, ID: e.ID}
	}), nil
}

type CreateEventInput struct {
	ApplicationID *uuid.UUID
	Type          domain.CalendarEventType
	Title         string
	StartsAt      time.Time
	EndsAt        time.Time
	TimeZone      string
	Notes         string
}

func (s *CalendarService) Create(ctx context.Context, userID uuid.UUID, in CreateEventInput) (*domain.CalendarEvent, error) {
	if err := validateEventInput(in.Type, in.Title, in.StartsAt, in.EndsAt, in.TimeZone); err != nil {
		return nil, err
	}
	if in.ApplicationID != nil {
		if _, err := s.applications.Get(ctx, userID, *in.ApplicationID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.NotFound()
			}
			return nil, domain.Internal()
		}
	}
	tz := in.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	now := time.Now().UTC()
	e := &domain.CalendarEvent{
		ID: uuid.New(), UserID: userID, Revision: 1,
		ApplicationID: in.ApplicationID, Type: in.Type, Title: in.Title,
		StartsAt: in.StartsAt, EndsAt: in.EndsAt, TimeZone: tz,
		Source: domain.EventSourceLocal, Notes: in.Notes,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.events.CreateCalendarEvent(ctx, e); err != nil {
		return nil, domain.Internal()
	}
	return e, nil
}

func validateEventInput(t domain.CalendarEventType, title string, startsAt, endsAt time.Time, tz string) *domain.Error {
	if !validEventType(t) {
		return domain.ValidationField("type", "unsupported event type")
	}
	if title == "" || len(title) > 300 {
		return domain.ValidationField("title", "title must be 1-300 characters")
	}
	if startsAt.IsZero() {
		return domain.ValidationField("startsAt", "required")
	}
	if !endsAt.After(startsAt) {
		return domain.ValidationField("endsAt", "must be after startsAt")
	}
	if tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			return domain.ValidationField("timeZone", "invalid IANA time zone")
		}
	}
	return nil
}

type PatchEventInput struct {
	ExpectedRevision int64
	Title            *string
	StartsAt         *time.Time
	EndsAt           *time.Time
	TimeZone         *string
	Notes            *string
}

func (s *CalendarService) Patch(ctx context.Context, userID, id uuid.UUID, in PatchEventInput) (*domain.CalendarEvent, error) {
	e, err := s.events.GetCalendarEvent(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	if e.Revision != in.ExpectedRevision {
		return nil, domain.RevisionConflict(e.Revision)
	}
	if in.Title != nil {
		if *in.Title == "" || len(*in.Title) > 300 {
			return nil, domain.ValidationField("title", "title must be 1-300 characters")
		}
		e.Title = *in.Title
	}
	if in.StartsAt != nil {
		e.StartsAt = *in.StartsAt
	}
	if in.EndsAt != nil {
		e.EndsAt = *in.EndsAt
	}
	if !e.EndsAt.After(e.StartsAt) {
		return nil, domain.ValidationField("endsAt", "must be after startsAt")
	}
	if in.TimeZone != nil {
		if _, err := time.LoadLocation(*in.TimeZone); err != nil {
			return nil, domain.ValidationField("timeZone", "invalid IANA time zone")
		}
		e.TimeZone = *in.TimeZone
	}
	if in.Notes != nil {
		e.Notes = *in.Notes
	}
	e.UpdatedAt = time.Now().UTC()
	if err := s.events.PatchCalendarEvent(ctx, e, in.ExpectedRevision); err != nil {
		return nil, mapRevisionErr(err)
	}
	e.Revision = in.ExpectedRevision + 1
	return e, nil
}

func (s *CalendarService) Delete(ctx context.Context, userID, id uuid.UUID, expectedRevision int64) error {
	if err := s.events.DeleteCalendarEvent(ctx, userID, id, expectedRevision); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NotFound()
		}
		return mapRevisionErr(err)
	}
	return nil
}
