package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type ApplicationService struct {
	applications domain.ApplicationStore
	jobs         domain.JobStore
	documents    domain.DocumentStore
	analyses     domain.GapAnalysisStore
	evidence     domain.EvidenceStore
	interviews   domain.InterviewStore
	submissions  domain.SubmissionStore
	approvals    *ApprovalService
	uow          domain.UnitOfWork
	adapters     map[string]bool
}

func NewApplicationService(applications domain.ApplicationStore, jobs domain.JobStore, documents domain.DocumentStore, analyses domain.GapAnalysisStore, evidence domain.EvidenceStore, interviews domain.InterviewStore, submissions domain.SubmissionStore, approvals *ApprovalService, uow domain.UnitOfWork, enabledAdapters map[string]bool) *ApplicationService {
	return &ApplicationService{
		applications: applications, jobs: jobs, documents: documents, analyses: analyses,
		evidence: evidence, interviews: interviews, submissions: submissions,
		approvals: approvals, uow: uow, adapters: enabledAdapters,
	}
}

func (s *ApplicationService) List(ctx context.Context, userID uuid.UUID, f domain.ApplicationFilter) (domain.Page[domain.Application], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.applications.List(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.Application]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(a domain.Application) domain.Cursor {
		return domain.Cursor{CreatedAt: a.CreatedAt, ID: a.ID}
	}), nil
}

func (s *ApplicationService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Application, error) {
	a, err := s.applications.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return a, nil
}

func (s *ApplicationService) Create(ctx context.Context, userID uuid.UUID, jobID uuid.UUID, notes string) (*domain.Application, error) {
	j, err := s.jobs.Get(ctx, userID, jobID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	now := time.Now().UTC()
	a := &domain.Application{
		ID: uuid.New(), UserID: userID, Revision: 1,
		JobID: jobID, Company: j.Company, Title: j.Title,
		Stage: domain.StageDiscovered, Notes: notes,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.applications.Create(ctx, a); err != nil {
		return nil, domain.Internal()
	}
	return a, nil
}

type PatchApplicationInput struct {
	ExpectedRevision int64
	Stage            *domain.ApplicationStage
	Notes            *string
	NextActionAt     *time.Time
	ClearNextAction  bool
}

func (s *ApplicationService) Patch(ctx context.Context, userID, id uuid.UUID, in PatchApplicationInput) (*domain.Application, error) {
	var out *domain.Application
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		a, err := s.applications.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if a.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(a.Revision)
		}
		now := time.Now().UTC()
		if in.Stage != nil {
			to := *in.Stage
			if !domain.ValidStage(to) {
				return domain.ValidationField("stage", "unsupported stage")
			}
			if to == domain.StageApplied && a.Stage != domain.StageApplied {
				return domain.InvalidTransition("APPLIED can only be reached through submissions")
			}
			if !domain.CanPatchStage(a.Stage, to) {
				return domain.InvalidTransition("cannot move from " + string(a.Stage) + " to " + string(to))
			}
			if to != a.Stage {
				from := a.Stage
				a.Stage = to
				if to == domain.StageApplied {
					a.AppliedAt = &now
				}
				if err := s.applications.AddEvent(ctx, &domain.ApplicationEvent{
					ID: uuid.New(), UserID: userID, ApplicationID: a.ID,
					Type:      domain.EventStageChanged,
					Payload:   map[string]any{"from": string(from), "to": string(to)},
					CreatedAt: now,
				}); err != nil {
					return err
				}
			}
		}
		if in.Notes != nil {
			a.Notes = *in.Notes
		}
		if in.ClearNextAction {
			a.NextActionAt = nil
		} else if in.NextActionAt != nil {
			a.NextActionAt = in.NextActionAt
		}
		a.UpdatedAt = now
		if err := s.applications.Update(ctx, a, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		a.Revision = in.ExpectedRevision + 1
		out = a
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type ImportApplicationInput struct {
	JobID     uuid.UUID
	Stage     domain.ApplicationStage
	AppliedAt time.Time
	Notes     string
	Confirmed bool
}

func (s *ApplicationService) Import(ctx context.Context, userID uuid.UUID, in ImportApplicationInput) (*domain.Application, error) {
	if !in.Confirmed {
		return nil, domain.ValidationField("confirmed", "must be true to record a past application")
	}
	if !domain.ValidStage(in.Stage) || in.Stage == domain.StageDiscovered {
		return nil, domain.ValidationField("stage", "unsupported stage for import")
	}
	j, err := s.jobs.Get(ctx, userID, in.JobID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	now := time.Now().UTC()
	a := &domain.Application{
		ID: uuid.New(), UserID: userID, Revision: 1,
		JobID: in.JobID, Company: j.Company, Title: j.Title,
		Stage: in.Stage, Notes: in.Notes, AppliedAt: &in.AppliedAt,
		Imported: true, CreatedAt: now, UpdatedAt: now,
	}
	err = s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.applications.Create(ctx, a); err != nil {
			return err
		}
		return s.applications.AddEvent(ctx, &domain.ApplicationEvent{
			ID: uuid.New(), UserID: userID, ApplicationID: a.ID,
			Type:      domain.EventImported,
			Payload:   map[string]any{"stage": string(in.Stage), "appliedAt": in.AppliedAt},
			CreatedAt: now,
		})
	})
	if err != nil {
		return nil, domain.Internal()
	}
	return a, nil
}

func (s *ApplicationService) Timeline(ctx context.Context, userID, id uuid.UUID, page domain.PageRequest) (domain.Page[domain.ApplicationEvent], error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return domain.Page[domain.ApplicationEvent]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.applications.ListEvents(ctx, userID, id, page, limit+1)
	if err != nil {
		return domain.Page[domain.ApplicationEvent]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(e domain.ApplicationEvent) domain.Cursor {
		return domain.Cursor{CreatedAt: e.CreatedAt, ID: e.ID}
	}), nil
}

type ChecklistItem struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Completed bool   `json:"completed"`
}

type Checklist struct {
	Items             []ChecklistItem `json:"items"`
	AutomationEnabled bool            `json:"automationEnabled"`
}

func (s *ApplicationService) Checklist(ctx context.Context, userID, id uuid.UUID) (*Checklist, error) {
	a, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	evidenceCount, _ := s.evidence.Count(ctx, userID)
	analysisCount, _ := s.analyses.CountByApplication(ctx, userID, a.ID)
	docCount, _ := s.documents.CountFinalizedByApplication(ctx, userID, a.ID)
	interviewCount, _ := s.interviews.CountByApplication(ctx, userID, a.ID)
	return &Checklist{
		Items: []ChecklistItem{
			{Key: "evidence", Label: "Career evidence collected", Completed: evidenceCount > 0},
			{Key: "analysis", Label: "Gap analysis completed", Completed: analysisCount > 0},
			{Key: "document_finalized", Label: "Document finalized", Completed: docCount > 0},
			{Key: "applied", Label: "Application submitted", Completed: a.AppliedAt != nil || domain.TerminalStage(a.Stage) || a.Stage == domain.StageApplied || stageAfterApplied(a.Stage)},
			{Key: "interview_prep", Label: "Interview session scheduled", Completed: interviewCount > 0},
		},
		AutomationEnabled: false,
	}, nil
}

func stageAfterApplied(s domain.ApplicationStage) bool {
	switch s {
	case domain.StageScreening, domain.StageInterview, domain.StageOffer:
		return true
	}
	return false
}

type CreateDraftInput struct {
	ExpectedRevision   int64
	Mode               domain.SubmissionMode
	Adapter            *string
	DocumentVersionIDs []uuid.UUID
	ConfirmedSubmitted bool
}

func (s *ApplicationService) CreateDraft(ctx context.Context, userID, appID uuid.UUID, in CreateDraftInput) (*domain.SubmissionDraft, error) {
	var out *domain.SubmissionDraft
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		a, err := s.applications.Get(ctx, userID, appID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if a.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(a.Revision)
		}
		if in.Mode != domain.SubmissionManual && in.Mode != domain.SubmissionAdapter {
			return domain.ValidationField("mode", "must be MANUAL_RECORD or ADAPTER")
		}
		if in.Mode == domain.SubmissionAdapter {
			if in.Adapter == nil || !s.adapters[*in.Adapter] {
				return domain.FeatureDisabled("adapter submission is not enabled for this site")
			}
		} else if in.Adapter != nil {
			return domain.ValidationField("adapter", "adapter is only valid with ADAPTER mode")
		}
		if len(in.DocumentVersionIDs) == 0 {
			return domain.ValidationField("documentVersionIds", "at least one document version is required")
		}
		for _, vid := range in.DocumentVersionIDs {
			v, err := s.documents.GetVersion(ctx, userID, uuid.Nil, vid)
			if err != nil {
				return domain.ValidationField("documentVersionIds", "version "+vid.String()+" not found")
			}
			if v.ApplicationID != appID {
				return domain.ValidationField("documentVersionIds", "version "+vid.String()+" belongs to another application")
			}
			doc, err := s.documents.Get(ctx, userID, v.DocumentID)
			if err != nil {
				return domain.Internal()
			}
			if doc.FinalizedVersionID == nil || *doc.FinalizedVersionID != vid {
				return domain.DocumentNotFinalized("document version " + vid.String() + " is not finalized")
			}
		}
		d := &domain.SubmissionDraft{
			ID: uuid.New(), UserID: userID, ApplicationID: appID,
			ApplicationRevision: a.Revision, Mode: in.Mode, Adapter: in.Adapter,
			DocumentVersionIDs: in.DocumentVersionIDs, ConfirmedSubmitted: in.ConfirmedSubmitted,
			PayloadHash: domain.HashJSON(map[string]any{
				"applicationId": appID, "applicationRevision": a.Revision,
				"mode": in.Mode, "adapter": in.Adapter, "documentVersionIds": in.DocumentVersionIDs,
			}),
			CreatedAt: time.Now().UTC(),
		}
		if err := s.submissions.CreateDraft(ctx, d); err != nil {
			return err
		}
		out = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type SubmitInput struct {
	ExpectedRevision int64
	DraftID          uuid.UUID
	ApprovalID       uuid.UUID
}

func (s *ApplicationService) Submit(ctx context.Context, userID, appID uuid.UUID, in SubmitInput) (*domain.Submission, error) {
	var out *domain.Submission
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		a, err := s.applications.Get(ctx, userID, appID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if a.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(a.Revision)
		}
		d, err := s.submissions.GetDraft(ctx, userID, in.DraftID)
		if err != nil {
			return domain.ValidationField("draftId", "draft not found")
		}
		if d.ApplicationID != appID {
			return domain.ValidationField("draftId", "draft belongs to another application")
		}
		if d.ApplicationRevision != a.Revision {
			return domain.ApprovalStale("application changed since the draft was created; create a new draft and approval")
		}
		if _, err := s.approvals.Consume(ctx, userID, in.ApprovalID, domain.ApprovalApplicationSubmit, appID, d.ID, d.PayloadHash); err != nil {
			return err
		}
		if d.Mode == domain.SubmissionManual && !d.ConfirmedSubmitted {
			return domain.ValidationField("confirmedSubmitted", "manual submission requires confirming the external submission completed")
		}
		now := time.Now().UTC()
		sub := &domain.Submission{
			ID: uuid.New(), UserID: userID, ApplicationID: appID,
			Mode: d.Mode, Adapter: d.Adapter, Status: domain.SubmissionSucceeded,
			DocumentVersionIDs: d.DocumentVersionIDs, ApprovalID: in.ApprovalID,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.submissions.Create(ctx, sub); err != nil {
			return err
		}
		if domain.TerminalStage(a.Stage) {
			return domain.InvalidTransition("application is closed")
		}
		if !domain.TerminalStage(a.Stage) && a.Stage != domain.StageApplied && !stageAfterApplied(a.Stage) {
			from := a.Stage
			a.Stage = domain.StageApplied
			a.AppliedAt = &now
			a.UpdatedAt = now
			if err := s.applications.Update(ctx, a, in.ExpectedRevision); err != nil {
				return mapRevisionErr(err)
			}
			a.Revision = in.ExpectedRevision + 1
			if err := s.applications.AddEvent(ctx, &domain.ApplicationEvent{
				ID: uuid.New(), UserID: userID, ApplicationID: a.ID,
				Type:      domain.EventStageChanged,
				Payload:   map[string]any{"from": string(from), "to": string(domain.StageApplied)},
				CreatedAt: now,
			}); err != nil {
				return err
			}
		}
		if err := s.applications.AddEvent(ctx, &domain.ApplicationEvent{
			ID: uuid.New(), UserID: userID, ApplicationID: a.ID,
			Type:      domain.EventSubmission,
			Payload:   map[string]any{"submissionId": sub.ID, "mode": d.Mode, "status": sub.Status},
			CreatedAt: now,
		}); err != nil {
			return err
		}
		out = sub
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ApplicationService) ListSubmissions(ctx context.Context, userID, appID uuid.UUID, page domain.PageRequest) (domain.Page[domain.Submission], error) {
	if _, err := s.Get(ctx, userID, appID); err != nil {
		return domain.Page[domain.Submission]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.submissions.List(ctx, userID, appID, page, limit+1)
	if err != nil {
		return domain.Page[domain.Submission]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(s domain.Submission) domain.Cursor {
		return domain.Cursor{CreatedAt: s.CreatedAt, ID: s.ID}
	}), nil
}
