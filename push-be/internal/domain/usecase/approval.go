package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type ApprovalService struct {
	approvals    domain.ApprovalStore
	applications domain.ApplicationStore
	evidence     domain.EvidenceStore
	documents    domain.DocumentStore
	submissions  domain.SubmissionStore
	projects     domain.ProjectStore
	uow          domain.UnitOfWork
}

func NewApprovalService(approvals domain.ApprovalStore, applications domain.ApplicationStore, evidence domain.EvidenceStore, documents domain.DocumentStore, submissions domain.SubmissionStore, projects domain.ProjectStore, uow domain.UnitOfWork) *ApprovalService {
	return &ApprovalService{
		approvals: approvals, applications: applications, evidence: evidence,
		documents: documents, submissions: submissions, projects: projects, uow: uow,
	}
}

type CreateApprovalInput struct {
	Kind           domain.ApprovalKind
	ApplicationID  uuid.UUID
	TargetID       uuid.UUID
	TargetRevision *int64
}

func (s *ApprovalService) List(ctx context.Context, userID uuid.UUID, f domain.ApprovalFilter) (domain.Page[domain.Approval], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.approvals.List(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.Approval]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(a domain.Approval) domain.Cursor {
		return domain.Cursor{CreatedAt: a.CreatedAt, ID: a.ID}
	}), nil
}

func (s *ApprovalService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Approval, map[string]any, error) {
	a, err := s.approvals.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.NotFound()
		}
		return nil, nil, domain.Internal()
	}
	summary, err := s.targetSummary(ctx, userID, a)
	if err != nil {
		return nil, nil, err
	}
	return a, summary, nil
}

func (s *ApprovalService) targetSummary(ctx context.Context, userID uuid.UUID, a *domain.Approval) (map[string]any, error) {
	switch a.Kind {
	case domain.ApprovalEvidenceUse:
		e, err := s.evidence.Get(ctx, userID, a.TargetID)
		if err != nil {
			return nil, domain.Internal()
		}
		return map[string]any{"type": "EVIDENCE", "id": e.ID, "title": e.Title, "kind": e.Kind}, nil
	case domain.ApprovalDocumentFinalize:
		v, err := s.documents.GetVersion(ctx, userID, uuid.Nil, a.TargetID)
		if err != nil {
			return nil, domain.Internal()
		}
		return map[string]any{"type": "DOCUMENT_VERSION", "id": v.ID, "documentId": v.DocumentID, "number": v.Number}, nil
	case domain.ApprovalApplicationSubmit:
		d, err := s.submissions.GetDraft(ctx, userID, a.TargetID)
		if err != nil {
			return nil, domain.Internal()
		}
		return map[string]any{"type": "SUBMISSION_DRAFT", "id": d.ID, "mode": d.Mode, "documentVersionIds": d.DocumentVersionIDs}, nil
	case domain.ApprovalCliExecute:
		r, err := s.projects.GetRun(ctx, userID, uuid.Nil, a.TargetID)
		if err != nil {
			return nil, domain.Internal()
		}
		return map[string]any{
			"type": "CLI_RUN", "id": r.ID, "provider": r.Provider,
			"workingDirectory": r.WorkingDirectory, "executable": r.Executable,
			"arguments": r.Arguments, "prompt": r.Prompt,
		}, nil
	}
	return map[string]any{}, nil
}

func (s *ApprovalService) Create(ctx context.Context, userID uuid.UUID, in CreateApprovalInput) (*domain.Approval, error) {
	if !domain.ValidApprovalKind(in.Kind) {
		return nil, domain.ValidationField("kind", "unsupported kind")
	}
	if _, err := s.applications.Get(ctx, userID, in.ApplicationID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	hash, targetRevision, err := s.hashTarget(ctx, userID, in)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expires := now.Add(domain.DefaultApprovalTTL)
	a := &domain.Approval{
		ID: uuid.New(), UserID: userID, Revision: 1,
		Kind: in.Kind, ApplicationID: in.ApplicationID, TargetID: in.TargetID,
		TargetRevision: targetRevision, PayloadHash: hash,
		Status: domain.ApprovalPending, ExpiresAt: &expires,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.approvals.Create(ctx, a); err != nil {
		return nil, domain.Internal()
	}
	return a, nil
}

func (s *ApprovalService) hashTarget(ctx context.Context, userID uuid.UUID, in CreateApprovalInput) (string, *int64, error) {
	switch in.Kind {
	case domain.ApprovalEvidenceUse:
		e, err := s.evidence.Get(ctx, userID, in.TargetID)
		if err != nil {
			return "", nil, domain.NotFound()
		}
		if e.Archived {
			return "", nil, domain.InvalidTransition("cannot approve archived evidence for new use")
		}
		return domain.HashJSON(map[string]any{
			"kind": in.Kind, "evidenceId": e.ID, "contentHash": e.Provenance.ContentHash,
		}), in.TargetRevision, nil
	case domain.ApprovalDocumentFinalize:
		v, err := s.documents.GetVersion(ctx, userID, uuid.Nil, in.TargetID)
		if err != nil {
			return "", nil, domain.NotFound()
		}
		if v.ApplicationID != in.ApplicationID {
			return "", nil, domain.NotFound()
		}
		return domain.HashJSON(map[string]any{
			"kind": in.Kind, "versionId": v.ID, "content": json.RawMessage(v.Content), "blocks": v.Blocks,
		}), in.TargetRevision, nil
	case domain.ApprovalApplicationSubmit:
		d, err := s.submissions.GetDraft(ctx, userID, in.TargetID)
		if err != nil {
			return "", nil, domain.NotFound()
		}
		if d.ApplicationID != in.ApplicationID {
			return "", nil, domain.NotFound()
		}
		return d.PayloadHash, in.TargetRevision, nil
	case domain.ApprovalCliExecute:
		r, err := s.projects.GetRun(ctx, userID, uuid.Nil, in.TargetID)
		if err != nil {
			return "", nil, domain.NotFound()
		}
		if r.ApplicationID != in.ApplicationID {
			return "", nil, domain.NotFound()
		}
		return r.PayloadHash, in.TargetRevision, nil
	}
	return "", nil, domain.ValidationField("kind", "unsupported kind")
}

func (s *ApprovalService) Decide(ctx context.Context, userID, id uuid.UUID, expectedRevision int64, decision domain.ApprovalStatus) (*domain.Approval, error) {
	if decision != domain.ApprovalApproved && decision != domain.ApprovalDenied {
		return nil, domain.ValidationField("decision", "must be APPROVED or DENIED")
	}
	var out *domain.Approval
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		a, err := s.approvals.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if a.Revision != expectedRevision {
			return domain.RevisionConflict(a.Revision)
		}
		now := time.Now().UTC()
		if a.Status == domain.ApprovalPending && a.Expired(now) {
			a.Status = domain.ApprovalExpired
			a.UpdatedAt = now
			if uerr := s.approvals.Update(ctx, a, expectedRevision); uerr != nil {
				return mapRevisionErr(uerr)
			}
			a.Revision = expectedRevision + 1
			return domain.InvalidTransition("approval expired")
		}
		if a.Status != domain.ApprovalPending {
			return domain.InvalidTransition("approval already decided")
		}
		a.Status = decision
		a.DecidedAt = &now
		a.UpdatedAt = now
		if err := s.approvals.Update(ctx, a, expectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		a.Revision = expectedRevision + 1
		out = a
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ApprovalService) Consume(ctx context.Context, userID, approvalID uuid.UUID, kind domain.ApprovalKind, applicationID, targetID uuid.UUID, expectedHash string) (*domain.Approval, error) {
	a, err := s.approvals.Get(ctx, userID, approvalID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ApprovalRequired("approval not found")
		}
		return nil, domain.Internal()
	}
	if a.Kind != kind || a.ApplicationID != applicationID || a.TargetID != targetID {
		return nil, domain.ApprovalRequired("approval does not cover this target")
	}
	now := time.Now().UTC()
	if a.Expired(now) {
		if a.Status == domain.ApprovalPending || a.Status == domain.ApprovalApproved {
			a.Status = domain.ApprovalExpired
			a.UpdatedAt = now
			_ = s.approvals.Update(ctx, a, a.Revision)
		}
		return nil, domain.ApprovalRequired("approval expired")
	}
	if a.Status != domain.ApprovalApproved {
		return nil, domain.ApprovalRequired("approval is not approved")
	}
	if expectedHash != "" && a.PayloadHash != expectedHash {
		return nil, domain.ApprovalStale("approved content has changed")
	}
	if domain.OneShotApproval(kind) {
		a.Status = domain.ApprovalConsumed
		a.ConsumedAt = &now
		a.UpdatedAt = now
		if err := s.approvals.Update(ctx, a, a.Revision); err != nil {
			return nil, mapRevisionErr(err)
		}
		a.Revision++
	}
	return a, nil
}

func (s *ApprovalService) RequireEvidenceUse(ctx context.Context, userID, applicationID, evidenceID uuid.UUID) error {
	a, err := s.approvals.FindActive(ctx, userID, domain.ApprovalEvidenceUse, applicationID, evidenceID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ApprovalRequired("EVIDENCE_USE approval required for evidence " + evidenceID.String())
		}
		return domain.Internal()
	}
	if a.Status != domain.ApprovalApproved {
		return domain.ApprovalRequired("EVIDENCE_USE approval required for evidence " + evidenceID.String())
	}
	return nil
}
