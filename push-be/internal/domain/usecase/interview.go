package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type InterviewService struct {
	interviews   domain.InterviewStore
	applications domain.ApplicationStore
	jobs         domain.JobStore
	evidence     domain.EvidenceStore
	ops          domain.OperationStore
	uow          domain.UnitOfWork
	ai           *AIGate
}

func NewInterviewService(interviews domain.InterviewStore, applications domain.ApplicationStore, jobs domain.JobStore, evidence domain.EvidenceStore, ops domain.OperationStore, uow domain.UnitOfWork, ai *AIGate) *InterviewService {
	return &InterviewService{interviews: interviews, applications: applications, jobs: jobs, evidence: evidence, ops: ops, uow: uow, ai: ai}
}

func (s *InterviewService) List(ctx context.Context, userID uuid.UUID, f domain.InterviewFilter) (domain.Page[domain.InterviewSession], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.interviews.List(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.InterviewSession]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(v domain.InterviewSession) domain.Cursor {
		return domain.Cursor{CreatedAt: v.CreatedAt, ID: v.ID}
	}), nil
}

func (s *InterviewService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.InterviewSession, error) {
	v, err := s.interviews.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return v, nil
}

func validateCompanySources(sources []domain.CompanySource, now time.Time) *domain.Error {
	if len(sources) > 10 {
		return domain.ValidationField("companySources", "at most 10 sources")
	}
	for i, cs := range sources {
		if err := domain.ValidateHTTPSURL(cs.SourceURL); err != nil {
			return domain.ValidationField("companySources", "sourceUrl must be http(s)")
		}
		if n := domain.CodePointLen(cs.SourceText); n < 1 || n > 20000 {
			return domain.ValidationField("companySources", "sourceText must be 1-20000 characters")
		}
		if cs.AccessedAt.After(now) {
			return domain.ValidationField("companySources", "accessedAt cannot be in the future")
		}
		_ = i
	}
	return nil
}

type CreateInterviewInput struct {
	ApplicationID   uuid.UUID
	Title           string
	ScheduledAt     time.Time
	DurationMinutes *int
	EvidenceIDs     []uuid.UUID
	CompanySources  []domain.CompanySource
	Notes           string
	TimeZone        string
}

func (s *InterviewService) Create(ctx context.Context, userID uuid.UUID, in CreateInterviewInput) (*domain.InterviewSession, error) {
	if in.Title == "" || len(in.Title) > 300 {
		return nil, domain.ValidationField("title", "title must be 1-300 characters")
	}
	if in.ScheduledAt.IsZero() {
		return nil, domain.ValidationField("scheduledAt", "required")
	}
	now := time.Now().UTC()
	if err := validateCompanySources(in.CompanySources, now); err != nil {
		return nil, err
	}
	var out *domain.InterviewSession
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		app, err := s.applications.Get(ctx, userID, in.ApplicationID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		for _, eid := range in.EvidenceIDs {
			if _, err := s.evidence.Get(ctx, userID, eid); err != nil {
				return domain.ValidationField("evidenceIds", "evidence "+eid.String()+" not found")
			}
		}
		tz := in.TimeZone
		if tz == "" {
			tz = "UTC"
		}
		if _, err := time.LoadLocation(tz); err != nil {
			return domain.ValidationField("timeZone", "invalid IANA time zone")
		}
		dur := in.DurationMinutes
		ends := in.ScheduledAt
		if dur != nil {
			ends = in.ScheduledAt.Add(time.Duration(*dur) * time.Minute)
		}
		event := &domain.CalendarEvent{
			ID: uuid.New(), UserID: userID, Revision: 1,
			ApplicationID: &app.ID, Type: domain.CalInterview,
			Title: in.Title, StartsAt: in.ScheduledAt, EndsAt: ends,
			TimeZone: tz, Source: domain.EventSourceLocal, Notes: in.Notes,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.interviews.CreateCalendarEvent(ctx, event); err != nil {
			return err
		}
		session := &domain.InterviewSession{
			ID: uuid.New(), UserID: userID, Revision: 1,
			ApplicationID: in.ApplicationID, Title: in.Title,
			ScheduledAt: in.ScheduledAt, DurationMinutes: dur, EventID: &event.ID,
			EvidenceIDs: nonNilUUIDs(in.EvidenceIDs), CompanySources: in.CompanySources,
			Notes: in.Notes, CreatedAt: now, UpdatedAt: now,
		}
		if session.CompanySources == nil {
			session.CompanySources = []domain.CompanySource{}
		}
		if err := s.interviews.Create(ctx, session); err != nil {
			return err
		}
		if err := s.applications.AddEvent(ctx, &domain.ApplicationEvent{
			ID: uuid.New(), UserID: userID, ApplicationID: app.ID,
			Type:      domain.EventInterview,
			Payload:   map[string]any{"interviewId": session.ID, "scheduledAt": in.ScheduledAt},
			CreatedAt: now,
		}); err != nil {
			return err
		}
		out = session
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func nonNilUUIDs(v []uuid.UUID) []uuid.UUID {
	if v == nil {
		return []uuid.UUID{}
	}
	return v
}

type PatchInterviewInput struct {
	ExpectedRevision int64
	Title            *string
	ScheduledAt      *time.Time
	CompanySources   *[]domain.CompanySource
	Notes            *string
	Reflection       *string
}

func (s *InterviewService) Patch(ctx context.Context, userID, id uuid.UUID, in PatchInterviewInput) (*domain.InterviewSession, error) {
	var out *domain.InterviewSession
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		v, err := s.interviews.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if v.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(v.Revision)
		}
		now := time.Now().UTC()
		if in.Title != nil {
			if *in.Title == "" || len(*in.Title) > 300 {
				return domain.ValidationField("title", "title must be 1-300 characters")
			}
			v.Title = *in.Title
		}
		if in.ScheduledAt != nil {
			v.ScheduledAt = *in.ScheduledAt
		}
		if in.CompanySources != nil {
			if err := validateCompanySources(*in.CompanySources, now); err != nil {
				return err
			}
			v.CompanySources = *in.CompanySources
		}
		if in.Notes != nil {
			v.Notes = *in.Notes
		}
		if in.Reflection != nil {
			v.Reflection = *in.Reflection
		}
		v.UpdatedAt = now
		if err := s.interviews.Update(ctx, v, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		v.Revision = in.ExpectedRevision + 1
		if v.EventID != nil && in.ScheduledAt != nil {
			ends := v.ScheduledAt
			if v.DurationMinutes != nil {
				ends = v.ScheduledAt.Add(time.Duration(*v.DurationMinutes) * time.Minute)
			}
			_ = s.interviews.UpdateCalendarEvent(ctx, &domain.CalendarEvent{
				ID: *v.EventID, StartsAt: v.ScheduledAt, EndsAt: ends,
			})
		}
		out = v
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type InterviewQuestion struct {
	Question    string   `json:"question"`
	Requirement string   `json:"requirement"`
	EvidenceIDs []string `json:"evidenceIds"`
}

type StarAnswer struct {
	EvidenceIDs []string `json:"evidenceIds"`
	Situation   string   `json:"situation"`
	Task        string   `json:"task"`
	Action      string   `json:"action"`
	Result      string   `json:"result"`
	NeedsInput  []string `json:"needsInput"`
}

type ResearchItem struct {
	Claim              string `json:"claim"`
	SourceURL          string `json:"sourceUrl"`
	AccessedAt         string `json:"accessedAt"`
	VerificationStatus string `json:"verificationStatus"`
}

type PrepareResult struct {
	Questions   []InterviewQuestion `json:"questions"`
	StarAnswers []StarAnswer        `json:"starAnswers"`
	Research    []ResearchItem      `json:"research"`
}

func (s *InterviewService) Prepare(ctx context.Context, userID, id uuid.UUID, expectedRevision int64, ai *domain.AiOptions) (*domain.Operation, error) {
	if err := s.ai.Check(ctx, userID, ai); err != nil {
		return nil, err
	}
	var op *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		v, err := s.interviews.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if v.Revision != expectedRevision {
			return domain.RevisionConflict(v.Revision)
		}
		app, err := s.applications.Get(ctx, userID, v.ApplicationID)
		if err != nil {
			return domain.Internal()
		}
		job, err := s.jobs.Get(ctx, userID, app.JobID)
		if err != nil {
			return domain.Internal()
		}
		eids := []string{}
		for _, e := range v.EvidenceIDs {
			eids = append(eids, e.String())
		}
		questions := []InterviewQuestion{}
		for _, req := range job.Requirements {
			questions = append(questions, InterviewQuestion{
				Question:    "Describe your experience with: " + req,
				Requirement: req, EvidenceIDs: eids,
			})
		}
		starAnswers := []StarAnswer{}
		for _, eid := range v.EvidenceIDs {
			e, err := s.evidence.Get(ctx, userID, eid)
			if err != nil {
				return err
			}
			starAnswers = append(starAnswers, StarAnswer{
				EvidenceIDs: []string{eid.String()},
				Situation:   e.Title, Task: "", Action: "", Result: "",
				NeedsInput: []string{"task", "action", "result"},
			})
		}
		research := []ResearchItem{}
		for _, cs := range v.CompanySources {
			claim := cs.SourceText
			if domain.CodePointLen(claim) > 200 {
				t, _ := domain.CodePointSlice(claim, 0, 200)
				claim = t
			}
			research = append(research, ResearchItem{
				Claim: claim, SourceURL: cs.SourceURL,
				AccessedAt:         cs.AccessedAt.UTC().Format(time.RFC3339),
				VerificationStatus: string(domain.VerificationUserProvided),
			})
		}
		res, _ := json.Marshal(PrepareResult{Questions: questions, StarAnswers: starAnswers, Research: research})
		now := time.Now().UTC()
		op = &domain.Operation{
			ID: uuid.New(), UserID: userID, Type: domain.OpInterviewPrepare,
			ApplicationID: &v.ApplicationID, Status: domain.OpSucceeded,
			Result:    &domain.OperationResult{Kind: string(domain.OpInterviewPrepare), Value: res},
			CreatedAt: now, UpdatedAt: now,
		}
		return s.ops.Create(ctx, op)
	})
	if err != nil {
		return nil, err
	}
	return op, nil
}
