package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type EvidenceService struct {
	evidence domain.EvidenceStore
	ops      domain.OperationStore
	uow      domain.UnitOfWork
}

func NewEvidenceService(evidence domain.EvidenceStore, ops domain.OperationStore, uow domain.UnitOfWork) *EvidenceService {
	return &EvidenceService{evidence: evidence, ops: ops, uow: uow}
}

type CreateEvidenceInput struct {
	Kind         domain.EvidenceKind
	Title        string
	SourceText   string
	SourceURL    *string
	Skills       []string
	SupersedesID *uuid.UUID
}

func validateEvidenceInput(in *CreateEvidenceInput) *domain.Error {
	if !domain.ValidEvidenceKind(in.Kind) {
		return domain.ValidationField("kind", "unsupported kind")
	}
	if len(in.Title) < 1 || len(in.Title) > 200 {
		return domain.ValidationField("title", "title must be 1-200 characters")
	}
	if n := len(in.SourceText); n < 1 || n > 100000 {
		return domain.ValidationField("sourceText", "sourceText must be 1-100000 characters")
	}
	if len(in.Skills) > 100 {
		return domain.ValidationField("skills", "at most 100 skills")
	}
	for _, sk := range in.Skills {
		if len(sk) < 1 || len(sk) > 100 {
			return domain.ValidationField("skills", "each skill must be 1-100 characters")
		}
	}
	if in.SourceURL != nil {
		if err := domain.ValidateHTTPSURL(*in.SourceURL); err != nil {
			return domain.ValidationField("sourceUrl", "must be an http(s) URL without credentials")
		}
	}
	return nil
}

func (s *EvidenceService) List(ctx context.Context, userID uuid.UUID, f domain.EvidenceFilter) (domain.Page[domain.CareerEvidence], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.evidence.List(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.CareerEvidence]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(e domain.CareerEvidence) domain.Cursor {
		return domain.Cursor{CreatedAt: e.CreatedAt, ID: e.ID}
	}), nil
}

func (s *EvidenceService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.CareerEvidence, error) {
	e, err := s.evidence.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return e, nil
}

func (s *EvidenceService) Create(ctx context.Context, userID uuid.UUID, in CreateEvidenceInput) (*domain.CareerEvidence, error) {
	if err := validateEvidenceInput(&in); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	e := &domain.CareerEvidence{
		ID: uuid.New(), UserID: userID, Revision: 1,
		Kind: in.Kind, Title: in.Title, SourceText: in.SourceText, SourceURL: in.SourceURL,
		Skills: in.Skills, VerificationStatus: domain.VerificationUserProvided,
		Provenance:   domain.Provenance{ContentHash: domain.HashBytes([]byte(in.SourceText))},
		SupersedesID: in.SupersedesID, CreatedAt: now, UpdatedAt: now,
	}
	if e.Skills == nil {
		e.Skills = []string{}
	}
	if in.SupersedesID != nil {
		if _, err := s.evidence.Get(ctx, userID, *in.SupersedesID); err != nil {
			return nil, domain.ValidationField("supersedesId", "referenced evidence not found")
		}
	}
	if err := s.evidence.Create(ctx, e); err != nil {
		return nil, domain.Internal()
	}
	return e, nil
}

func (s *EvidenceService) Archive(ctx context.Context, userID, id uuid.UUID, expectedRevision int64) (*domain.CareerEvidence, error) {
	var out *domain.CareerEvidence
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		e, err := s.evidence.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if e.Archived {
			out = e
			return nil
		}
		e.Archived = true
		e.UpdatedAt = time.Now().UTC()
		if err := s.evidence.Update(ctx, e, expectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		e.Revision = expectedRevision + 1
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func mapRevisionErr(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return domain.NotFound()
	}
	if c, ok := domain.AsConflict(err); ok {
		return domain.RevisionConflict(c.Current)
	}
	return err
}

type ImportInput struct {
	SourceID    *uuid.UUID
	Text        string
	SourceURL   *string
	ContentHash string
	Format      string
}

func ValidImportFormat(f string) bool {
	switch f {
	case "TEXT", "MARKDOWN", "PDF", "DOCX", "GITHUB":
		return true
	}
	return false
}

func (s *EvidenceService) Import(ctx context.Context, userID uuid.UUID, in ImportInput) (*domain.Operation, error) {
	if !ValidImportFormat(in.Format) {
		return nil, domain.ValidationField("format", "unsupported format")
	}
	if in.Format == "GITHUB" {
		return nil, domain.IntegrationRequired("GitHub collection requires a connected integration")
	}
	if in.SourceURL != nil {
		if err := domain.ValidateHTTPSURL(*in.SourceURL); err != nil {
			return nil, domain.ValidationField("sourceUrl", "must be an http(s) URL without credentials")
		}
	}
	if in.SourceID != nil {
		return nil, domain.ValidationField("sourceId", "stored sources are not supported by this build; pass extracted text")
	}
	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]any{
		"text": in.Text, "sourceUrl": in.SourceURL, "contentHash": in.ContentHash, "format": in.Format,
	})
	op := &domain.Operation{
		ID: uuid.New(), UserID: userID, Type: domain.OpEvidenceImport,
		CreatedAt: now, UpdatedAt: now, PendingPayload: payload,
	}
	s.progressImport(op)
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		if err := s.persistImportResult(ctx, op); err != nil {
			return err
		}
		return s.ops.Create(ctx, op)
	})
	if err != nil {
		return nil, domain.Internal()
	}
	return op, nil
}

func (s *EvidenceService) persistImportResult(ctx context.Context, op *domain.Operation) error {
	if op.Status != domain.OpSucceeded || op.Result == nil {
		return nil
	}
	var res struct {
		Evidence []domain.CareerEvidence `json:"evidence"`
	}
	if err := json.Unmarshal(op.Result.Value, &res); err != nil || len(res.Evidence) == 0 {
		return nil
	}
	created := &res.Evidence[0]
	if err := s.evidence.Create(ctx, created); err != nil {
		return err
	}
	v, _ := json.Marshal(map[string]any{"evidence": []any{created}, "warnings": []any{}})
	op.Result = &domain.OperationResult{Kind: string(domain.OpEvidenceImport), Value: v}
	return nil
}

func (s *EvidenceService) progressImport(op *domain.Operation) {
	var p struct {
		Text        string  `json:"text"`
		SourceURL   *string `json:"sourceUrl"`
		ContentHash string  `json:"contentHash"`
		Format      string  `json:"format"`
		Kind        string  `json:"kind"`
		Title       string  `json:"title"`
	}
	_ = json.Unmarshal(op.PendingPayload, &p)
	missing := []domain.InputField{}
	if p.Text == "" {
		missing = append(missing, domain.InputField{Name: "text", Label: "Extracted source text", Type: "TEXT"})
	}
	if !domain.ValidEvidenceKind(domain.EvidenceKind(p.Kind)) {
		missing = append(missing, domain.InputField{Name: "kind", Label: "Evidence kind (RESUME|CAREER|EDUCATION|SKILL|PROJECT)", Type: "TEXT"})
	}
	if p.Title == "" {
		missing = append(missing, domain.InputField{Name: "title", Label: "Evidence title", Type: "TEXT"})
	}
	now := time.Now().UTC()
	if len(missing) > 0 {
		op.Status = domain.OpNeedsInput
		op.InputRequest = &domain.InputRequest{
			Code:    "IMPORT_DETAILS_REQUIRED",
			Message: "text, kind and title are required to create evidence",
			Fields:  missing,
		}
		op.UpdatedAt = now
		return
	}
	evidence := &domain.CareerEvidence{
		ID: uuid.New(), UserID: op.UserID, Revision: 1,
		Kind: domain.EvidenceKind(p.Kind), Title: p.Title, SourceText: p.Text,
		SourceURL: p.SourceURL, Skills: []string{},
		VerificationStatus: domain.VerificationUserProvided,
		Provenance: domain.Provenance{
			ContentHash:    firstNonEmpty(p.ContentHash, domain.HashBytes([]byte(p.Text))),
			SourceLocation: &domain.SourceLocation{Start: 0, End: domain.CodePointLen(p.Text), Unit: "CODE_POINT"},
		},
		CreatedAt: now, UpdatedAt: now,
	}
	op.Result = evidenceResult(evidence)
	op.Status = domain.OpSucceeded
	op.UpdatedAt = now
}

func evidenceResult(e *domain.CareerEvidence) *domain.OperationResult {
	v, _ := json.Marshal(map[string]any{
		"evidence": []any{e}, "warnings": []any{},
	})
	return &domain.OperationResult{Kind: string(domain.OpEvidenceImport), Value: v}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (s *EvidenceService) CompleteImportInput(ctx context.Context, op *domain.Operation, fields map[string]string) error {
	var p map[string]any
	if err := json.Unmarshal(op.PendingPayload, &p); err != nil {
		p = map[string]any{}
	}
	allowed := map[string]bool{}
	for _, f := range op.InputRequest.Fields {
		allowed[f.Name] = true
	}
	for k, v := range fields {
		if !allowed[k] {
			return domain.ValidationField("fields."+k, "field not requested")
		}
		p[k] = v
	}
	op.PendingPayload, _ = json.Marshal(p)
	s.progressImport(op)
	return s.persistImportResult(ctx, op)
}
