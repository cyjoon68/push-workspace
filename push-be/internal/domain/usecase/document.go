package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type DocumentService struct {
	documents    domain.DocumentStore
	applications domain.ApplicationStore
	evidence     domain.EvidenceStore
	approvals    *ApprovalService
	ops          domain.OperationStore
	uow          domain.UnitOfWork
	ai           *AIGate
}

func NewDocumentService(documents domain.DocumentStore, applications domain.ApplicationStore, evidence domain.EvidenceStore, approvals *ApprovalService, ops domain.OperationStore, uow domain.UnitOfWork, ai *AIGate) *DocumentService {
	return &DocumentService{documents: documents, applications: applications, evidence: evidence, approvals: approvals, ops: ops, uow: uow, ai: ai}
}

func (s *DocumentService) List(ctx context.Context, userID uuid.UUID, f domain.DocumentFilter) (domain.Page[domain.Document], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.documents.List(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.Document]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(d domain.Document) domain.Cursor {
		return domain.Cursor{CreatedAt: d.CreatedAt, ID: d.ID}
	}), nil
}

func (s *DocumentService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Document, error) {
	d, err := s.documents.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return d, nil
}

type CreateDocumentInput struct {
	ApplicationID uuid.UUID
	Title         string
	Kind          domain.DocumentKind
	Template      domain.DocumentTemplate
	Language      string
}

func (s *DocumentService) Create(ctx context.Context, userID uuid.UUID, in CreateDocumentInput) (*domain.Document, error) {
	if _, err := s.applications.Get(ctx, userID, in.ApplicationID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	if err := validateDocumentInput(in.Title, in.Kind, in.Template); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	d := &domain.Document{
		ID: uuid.New(), UserID: userID, Revision: 1,
		ApplicationID: in.ApplicationID, Title: in.Title, Kind: in.Kind,
		Template: in.Template, Language: in.Language, Status: domain.DocStatusDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.documents.Create(ctx, d); err != nil {
		return nil, domain.Internal()
	}
	return d, nil
}

func validateDocumentInput(title string, kind domain.DocumentKind, template domain.DocumentTemplate) *domain.Error {
	if len(title) < 1 || len(title) > 300 {
		return domain.ValidationField("title", "title must be 1-300 characters")
	}
	if !domain.ValidDocumentKind(kind) {
		return domain.ValidationField("kind", "must be RESUME, PORTFOLIO or COVER_LETTER")
	}
	if !domain.ValidDocumentTemplate(template) {
		return domain.ValidationField("template", "must be CLASSIC, MODERN or COMPACT")
	}
	return nil
}

type PatchDocumentInput struct {
	ExpectedRevision int64
	Title            *string
	Template         *domain.DocumentTemplate
	Language         *string
}

func (s *DocumentService) Patch(ctx context.Context, userID, id uuid.UUID, in PatchDocumentInput) (*domain.Document, error) {
	var out *domain.Document
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		d, err := s.getMutable(ctx, userID, id)
		if err != nil {
			return err
		}
		if d.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(d.Revision)
		}
		if in.Title != nil {
			if len(*in.Title) < 1 || len(*in.Title) > 300 {
				return domain.ValidationField("title", "title must be 1-300 characters")
			}
			d.Title = *in.Title
		}
		if in.Template != nil {
			if !domain.ValidDocumentTemplate(*in.Template) {
				return domain.ValidationField("template", "must be CLASSIC, MODERN or COMPACT")
			}
			if *in.Template != d.Template {
				d.Template = *in.Template
				d.FinalizedVersionID = nil
				if d.Status == domain.DocStatusFinalized {
					d.Status = domain.DocStatusDraft
				}
			}
		}
		if in.Language != nil {
			d.Language = *in.Language
		}
		d.UpdatedAt = time.Now().UTC()
		if err := s.documents.Update(ctx, d, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		d.Revision = in.ExpectedRevision + 1
		out = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DocumentService) getMutable(ctx context.Context, userID, id uuid.UUID) (*domain.Document, error) {
	d, err := s.documents.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	if d.Status == domain.DocStatusArchived {
		return nil, domain.InvalidTransition("document is archived")
	}
	return d, nil
}

func (s *DocumentService) verifyBlocks(ctx context.Context, userID, applicationID uuid.UUID, blocks []domain.Block) (map[uuid.UUID]*domain.CareerEvidence, *domain.Error) {
	loaded := map[uuid.UUID]*domain.CareerEvidence{}
	for i := range blocks {
		b := &blocks[i]
		if len(b.EvidenceRefs) == 0 {
			b.ClaimStatus = domain.ClaimUnsupported
			continue
		}
		status := domain.ClaimSupported
		for _, ref := range b.EvidenceRefs {
			e, ok := loaded[ref.EvidenceID]
			if !ok {
				var err error
				e, err = s.evidence.Get(ctx, userID, ref.EvidenceID)
				if err != nil {
					return nil, domain.ValidationField("blocks", "evidence "+ref.EvidenceID.String()+" not found")
				}
				loaded[ref.EvidenceID] = e
			}
			if e.Archived {
				return nil, domain.ValidationField("blocks", "evidence "+ref.EvidenceID.String()+" is archived")
			}
			span, ok := domain.CodePointSlice(e.SourceText, ref.Start, ref.End)
			if !ok {
				return nil, domain.ValidationField("blocks", "evidenceRef range out of bounds for "+ref.EvidenceID.String())
			}
			if span == "" || !strings.Contains(b.Text, span) {
				status = domain.ClaimNeedsReview
			}
		}
		b.ClaimStatus = status
	}
	return loaded, nil
}

func computeQuality(blocks []domain.Block, method domain.AnalysisMethod) domain.Quality {
	issues := []domain.QualityIssue{}
	unsupported := 0
	for _, b := range blocks {
		switch b.ClaimStatus {
		case domain.ClaimUnsupported:
			unsupported++
			issues = append(issues, domain.QualityIssue{Code: "UNSUPPORTED_CLAIM", Severity: "ERROR", BlockID: b.ID, Message: "block has no evidence references"})
		case domain.ClaimNeedsReview:
			issues = append(issues, domain.QualityIssue{Code: "NEEDS_REVIEW", Severity: "WARN", BlockID: b.ID, Message: "evidence span does not literally appear in block text"})
		}
	}
	var fidelity *int
	if len(blocks) > 0 {
		v := (len(blocks) - unsupported) * 100 / len(blocks)
		fidelity = &v
	}
	return domain.Quality{
		EvidenceFidelity: fidelity, Method: method, Issues: issues,
	}
}

type CreateVersionInput struct {
	ExpectedRevision int64
	Content          json.RawMessage
	Blocks           []domain.Block
	ChangeNote       string
}

func (s *DocumentService) CreateVersion(ctx context.Context, userID, docID uuid.UUID, in CreateVersionInput) (*domain.Document, *domain.DocumentVersion, error) {
	var outDoc *domain.Document
	var outVer *domain.DocumentVersion
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		d, err := s.getMutable(ctx, userID, docID)
		if err != nil {
			return err
		}
		if d.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(d.Revision)
		}
		if err := domain.ValidateDocumentContent(in.Content, in.Blocks); err != nil {
			return err
		}
		blocks := make([]domain.Block, len(in.Blocks))
		copy(blocks, in.Blocks)
		if _, err := s.verifyBlocks(ctx, userID, d.ApplicationID, blocks); err != nil {
			return err
		}
		num, err := s.documents.NextVersionNumber(ctx, docID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		v := &domain.DocumentVersion{
			ID: uuid.New(), UserID: userID, DocumentID: docID, ApplicationID: d.ApplicationID,
			Number: num, Content: in.Content, Blocks: blocks, ChangeNote: in.ChangeNote,
			Quality: computeQuality(blocks, domain.MethodRuleBased), CreatedAt: now,
		}
		if err := s.documents.CreateVersion(ctx, v); err != nil {
			return err
		}
		d.LatestVersionID = &v.ID
		d.FinalizedVersionID = nil
		if d.Status == domain.DocStatusFinalized {
			d.Status = domain.DocStatusDraft
		}
		d.UpdatedAt = now
		if err := s.documents.Update(ctx, d, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		d.Revision = in.ExpectedRevision + 1
		outDoc, outVer = d, v
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return outDoc, outVer, nil
}

func (s *DocumentService) ListVersions(ctx context.Context, userID, docID uuid.UUID, page domain.PageRequest) (domain.Page[domain.DocumentVersion], error) {
	if _, err := s.Get(ctx, userID, docID); err != nil {
		return domain.Page[domain.DocumentVersion]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.documents.ListVersions(ctx, userID, docID, page, limit+1)
	if err != nil {
		return domain.Page[domain.DocumentVersion]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(v domain.DocumentVersion) domain.Cursor {
		return domain.Cursor{CreatedAt: v.CreatedAt, ID: v.ID}
	}), nil
}

func (s *DocumentService) GetVersion(ctx context.Context, userID, docID, versionID uuid.UUID) (*domain.DocumentVersion, error) {
	if _, err := s.Get(ctx, userID, docID); err != nil {
		return nil, err
	}
	v, err := s.documents.GetVersion(ctx, userID, docID, versionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return v, nil
}

type GenerateInput struct {
	ExpectedRevision int64
	EvidenceIDs      []uuid.UUID
	AnalysisID       *uuid.UUID
	AI               *domain.AiOptions
	Language         *string
}

func (s *DocumentService) Generate(ctx context.Context, userID, docID uuid.UUID, in GenerateInput) (*domain.Operation, error) {
	if err := s.ai.Check(ctx, userID, in.AI); err != nil {
		return nil, err
	}
	var op *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		d, err := s.getMutable(ctx, userID, docID)
		if err != nil {
			return err
		}
		if d.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(d.Revision)
		}
		if len(in.EvidenceIDs) == 0 {
			return domain.ValidationField("evidenceIds", "at least one evidence is required")
		}
		entries := []domain.ExcerptEntry{}
		for _, eid := range in.EvidenceIDs {
			e, err := s.evidence.Get(ctx, userID, eid)
			if err != nil {
				return domain.ValidationField("evidenceIds", "evidence "+eid.String()+" not found")
			}
			if e.Archived {
				return domain.ValidationField("evidenceIds", "evidence "+eid.String()+" is archived")
			}
			if err := s.approvals.RequireEvidenceUse(ctx, userID, d.ApplicationID, eid); err != nil {
				return err
			}
			text := e.SourceText
			if domain.CodePointLen(text) > 500 {
				t, _ := domain.CodePointSlice(text, 0, 500)
				text = t
			}
			entries = append(entries, domain.ExcerptEntry{
				EvidenceID: eid, Text: text, Start: 0, End: domain.CodePointLen(text),
			})
		}
		if in.Language != nil {
			d.Language = *in.Language
		}
		num, err := s.documents.NextVersionNumber(ctx, docID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		content, blocks := domain.BuildExcerptContent(d.Title, entries)
		v := &domain.DocumentVersion{
			ID: uuid.New(), UserID: userID, DocumentID: docID, ApplicationID: d.ApplicationID,
			Number: num, Content: content, Blocks: blocks, ChangeNote: "generated",
			Quality: computeQuality(blocks, domain.MethodRuleBased), CreatedAt: now,
		}
		if err := s.documents.CreateVersion(ctx, v); err != nil {
			return err
		}
		d.LatestVersionID = &v.ID
		d.FinalizedVersionID = nil
		if d.Status == domain.DocStatusFinalized {
			d.Status = domain.DocStatusDraft
		}
		d.UpdatedAt = now
		if err := s.documents.Update(ctx, d, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		d.Revision = in.ExpectedRevision + 1
		res, _ := json.Marshal(map[string]any{"document": d, "version": v})
		op = &domain.Operation{
			ID: uuid.New(), UserID: userID, Type: domain.OpDocumentGenerate,
			ApplicationID: &d.ApplicationID, Status: domain.OpSucceeded,
			Result:    &domain.OperationResult{Kind: string(domain.OpDocumentGenerate), Value: res},
			CreatedAt: now, UpdatedAt: now,
		}
		return s.ops.Create(ctx, op)
	})
	if err != nil {
		return nil, err
	}
	return op, nil
}

type ReviseInput struct {
	ExpectedRevision int64
	VersionID        uuid.UUID
	Selection        domain.Selection
	Action           domain.RevisionAction
	Instruction      string
	AI               *domain.AiOptions
}

func (s *DocumentService) CreateRevision(ctx context.Context, userID, docID uuid.UUID, in ReviseInput) (*domain.Operation, error) {
	if err := s.ai.Check(ctx, userID, in.AI); err != nil {
		return nil, err
	}
	if !domain.ValidRevisionAction(in.Action) {
		return nil, domain.ValidationField("action", "unsupported action")
	}
	if in.Selection.From < 0 || in.Selection.To <= in.Selection.From || in.Selection.Text == "" {
		return nil, domain.ValidationField("selection", "requires from < to and matching text")
	}
	var op *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		d, err := s.getMutable(ctx, userID, docID)
		if err != nil {
			return err
		}
		if d.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(d.Revision)
		}
		v, err := s.documents.GetVersion(ctx, userID, docID, in.VersionID)
		if err != nil {
			return domain.NotFound()
		}
		found := false
		for _, b := range v.Blocks {
			if strings.Contains(b.Text, in.Selection.Text) {
				found = true
				break
			}
		}
		if !found {
			return domain.ValidationField("selection", "selection text does not match the version content")
		}
		now := time.Now().UTC()
		replacement := in.Selection.Text
		if in.Action == domain.RevShorten && len([]rune(replacement)) > 1 {
			r := []rune(replacement)
			replacement = string(r[:len(r)/2])
		}
		p := &domain.RevisionProposal{
			ID: uuid.New(), UserID: userID, DocumentID: docID,
			SourceVersionID: v.ID, SourceRevision: d.Revision,
			Selection: in.Selection, Replacement: replacement,
			EvidenceRefs: []domain.EvidenceRef{}, ClaimStatus: domain.ClaimNeedsReview,
			CreatedAt: now,
		}
		if err := s.documents.CreateProposal(ctx, p); err != nil {
			return err
		}
		res, _ := json.Marshal(p)
		op = &domain.Operation{
			ID: uuid.New(), UserID: userID, Type: domain.OpDocumentRevise,
			ApplicationID: &d.ApplicationID, Status: domain.OpSucceeded,
			Result:    &domain.OperationResult{Kind: string(domain.OpDocumentRevise), Value: res},
			CreatedAt: now, UpdatedAt: now,
		}
		return s.ops.Create(ctx, op)
	})
	if err != nil {
		return nil, err
	}
	return op, nil
}

func (s *DocumentService) ApplyRevision(ctx context.Context, userID, docID, proposalID uuid.UUID, expectedRevision int64) (*domain.Document, *domain.DocumentVersion, error) {
	var outDoc *domain.Document
	var outVer *domain.DocumentVersion
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		d, err := s.getMutable(ctx, userID, docID)
		if err != nil {
			return err
		}
		if d.Revision != expectedRevision {
			return domain.RevisionConflict(d.Revision)
		}
		p, err := s.documents.GetProposal(ctx, userID, docID, proposalID)
		if err != nil {
			return domain.NotFound()
		}
		if p.AppliedAt != nil {
			return domain.InvalidTransition("proposal already applied")
		}
		if d.LatestVersionID == nil || *d.LatestVersionID != p.SourceVersionID || p.SourceRevision != d.Revision {
			return domain.ApprovalStale("document changed since the proposal was created")
		}
		src, err := s.documents.GetVersion(ctx, userID, docID, p.SourceVersionID)
		if err != nil {
			return domain.Internal()
		}
		var newContent json.RawMessage
		applied := false
		var root domain.Node
		if err := json.Unmarshal(src.Content, &root); err != nil {
			return domain.Internal()
		}
		replaceInNode(&root, p.Selection.Text, p.Replacement, &applied)
		newContent, _ = json.Marshal(root)
		newBlocks := make([]domain.Block, len(src.Blocks))
		copy(newBlocks, src.Blocks)
		for i := range newBlocks {
			if strings.Contains(newBlocks[i].Text, p.Selection.Text) {
				newBlocks[i].Text = strings.Replace(newBlocks[i].Text, p.Selection.Text, p.Replacement, 1)
			}
		}
		if !applied {
			return domain.ApprovalStale("selection text no longer present in the version")
		}
		if err := domain.ValidateDocumentContent(newContent, newBlocks); err != nil {
			return err
		}
		if _, err := s.verifyBlocks(ctx, userID, d.ApplicationID, newBlocks); err != nil {
			return err
		}
		num, err := s.documents.NextVersionNumber(ctx, docID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		v := &domain.DocumentVersion{
			ID: uuid.New(), UserID: userID, DocumentID: docID, ApplicationID: d.ApplicationID,
			Number: num, Content: newContent, Blocks: newBlocks,
			ChangeNote: "applied proposal " + p.ID.String(),
			Quality:    computeQuality(newBlocks, domain.MethodRuleBased), CreatedAt: now,
		}
		if err := s.documents.CreateVersion(ctx, v); err != nil {
			return err
		}
		if err := s.documents.MarkProposalApplied(ctx, p.ID, now); err != nil {
			return err
		}
		d.LatestVersionID = &v.ID
		d.FinalizedVersionID = nil
		if d.Status == domain.DocStatusFinalized {
			d.Status = domain.DocStatusDraft
		}
		d.UpdatedAt = now
		if err := s.documents.Update(ctx, d, expectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		d.Revision = expectedRevision + 1
		outDoc, outVer = d, v
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return outDoc, outVer, nil
}

func replaceInNode(n *domain.Node, old, new string, applied *bool) {
	if *applied {
		return
	}
	if n.Type == "text" && strings.Contains(n.Text, old) {
		n.Text = strings.Replace(n.Text, old, new, 1)
		*applied = true
		return
	}
	for i := range n.Content {
		replaceInNode(&n.Content[i], old, new, applied)
	}
}

func (s *DocumentService) Review(ctx context.Context, userID, docID, versionID uuid.UUID) (*domain.DocumentVersion, error) {
	v, err := s.GetVersion(ctx, userID, docID, versionID)
	if err != nil {
		return nil, err
	}
	blocks := make([]domain.Block, len(v.Blocks))
	copy(blocks, v.Blocks)
	d, err := s.Get(ctx, userID, docID)
	if err != nil {
		return nil, err
	}
	if _, err := s.verifyBlocks(ctx, userID, d.ApplicationID, blocks); err != nil {
		return nil, err
	}
	v.Blocks = blocks
	v.Quality = computeQuality(blocks, domain.MethodRuleBased)
	return v, nil
}

func (s *DocumentService) versionHash(v *domain.DocumentVersion) string {
	return domain.HashJSON(map[string]any{
		"kind": domain.ApprovalDocumentFinalize, "versionId": v.ID,
		"content": json.RawMessage(v.Content), "blocks": v.Blocks,
	})
}

func (s *DocumentService) Finalize(ctx context.Context, userID, docID uuid.UUID, expectedRevision int64, versionID, approvalID uuid.UUID) (*domain.Document, error) {
	var out *domain.Document
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		d, err := s.getMutable(ctx, userID, docID)
		if err != nil {
			return err
		}
		if d.Revision != expectedRevision {
			return domain.RevisionConflict(d.Revision)
		}
		v, err := s.documents.GetVersion(ctx, userID, docID, versionID)
		if err != nil {
			return domain.NotFound()
		}
		blocks := make([]domain.Block, len(v.Blocks))
		copy(blocks, v.Blocks)
		refs, err := s.verifyBlocks(ctx, userID, d.ApplicationID, blocks)
		if err != nil {
			return err
		}
		for _, b := range blocks {
			if b.ClaimStatus != domain.ClaimSupported {
				return domain.UnsupportedClaim("block " + b.ID + " is " + string(b.ClaimStatus))
			}
		}
		for eid := range refs {
			if err := s.approvals.RequireEvidenceUse(ctx, userID, d.ApplicationID, eid); err != nil {
				return err
			}
		}
		v.Blocks = blocks
		if _, err := s.approvals.Consume(ctx, userID, approvalID, domain.ApprovalDocumentFinalize, d.ApplicationID, versionID, s.versionHash(v)); err != nil {
			return err
		}
		now := time.Now().UTC()
		d.Status = domain.DocStatusFinalized
		d.FinalizedVersionID = &versionID
		d.UpdatedAt = now
		if err := s.documents.Update(ctx, d, expectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		d.Revision = expectedRevision + 1
		if err := s.applications.AddEvent(ctx, &domain.ApplicationEvent{
			ID: uuid.New(), UserID: userID, ApplicationID: d.ApplicationID,
			Type:      domain.EventDocumentFinal,
			Payload:   map[string]any{"documentId": docID, "versionId": versionID},
			CreatedAt: now,
		}); err != nil {
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

func (s *DocumentService) Archive(ctx context.Context, userID, docID uuid.UUID, expectedRevision int64) (*domain.Document, error) {
	var out *domain.Document
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		d, err := s.documents.Get(ctx, userID, docID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if d.Revision != expectedRevision {
			return domain.RevisionConflict(d.Revision)
		}
		if d.Status == domain.DocStatusArchived {
			out = d
			return nil
		}
		d.Status = domain.DocStatusArchived
		d.UpdatedAt = time.Now().UTC()
		if err := s.documents.Update(ctx, d, expectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		d.Revision = expectedRevision + 1
		out = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type CreateExportInput struct {
	VersionID       uuid.UUID
	Format          domain.ExportFormat
	RendererVersion string
}

func (s *DocumentService) CreateExport(ctx context.Context, userID, docID uuid.UUID, in CreateExportInput) (*domain.DocumentExport, error) {
	if in.Format != domain.ExportPDF && in.Format != domain.ExportDOCX {
		return nil, domain.ValidationField("format", "must be PDF or DOCX")
	}
	if in.RendererVersion == "" {
		return nil, domain.ValidationField("rendererVersion", "required")
	}
	var out *domain.DocumentExport
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		d, err := s.documents.Get(ctx, userID, docID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		v, err := s.documents.GetVersion(ctx, userID, docID, in.VersionID)
		if err != nil {
			return domain.NotFound()
		}
		if d.FinalizedVersionID == nil || *d.FinalizedVersionID != v.ID {
			return domain.DocumentNotFinalized("only the finalized version can be exported")
		}
		now := time.Now().UTC()
		e := &domain.DocumentExport{
			ID: uuid.New(), UserID: userID, DocumentID: docID, VersionID: v.ID,
			Format: in.Format, Template: d.Template, Language: d.Language,
			Content: v.Content, Blocks: v.Blocks,
			ContentHash: domain.HashBytes(v.Content), Status: domain.ExportReadyToRender,
			RendererVersion: in.RendererVersion, CreatedAt: now, UpdatedAt: now,
		}
		if err := s.documents.CreateExport(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type ExportResultInput struct {
	SHA256     string
	ByteLength int64
	PageCount  *int
	Validation domain.ExportValidation
	Status     domain.ExportStatus
	ErrorCode  *string
}

func (s *DocumentService) RecordExportResult(ctx context.Context, userID, docID, exportID uuid.UUID, in ExportResultInput) (*domain.DocumentExport, error) {
	if in.Status != domain.ExportSucceeded && in.Status != domain.ExportFailed {
		return nil, domain.ValidationField("status", "must be SUCCEEDED or FAILED")
	}
	if in.SHA256 == "" || in.ByteLength < 0 {
		return nil, domain.ValidationField("sha256", "sha256 and non-negative byteLength are required")
	}
	var out *domain.DocumentExport
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		e, err := s.documents.GetExport(ctx, userID, docID, exportID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if e.Status != domain.ExportReadyToRender {
			return domain.InvalidTransition("export result already recorded")
		}
		e.Status = in.Status
		e.Result = &domain.ExportResult{
			SHA256: in.SHA256, ByteLength: in.ByteLength, PageCount: in.PageCount,
			Validation: in.Validation, ErrorCode: in.ErrorCode,
		}
		e.UpdatedAt = time.Now().UTC()
		if err := s.documents.UpdateExportResult(ctx, e); err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DocumentService) ListExports(ctx context.Context, userID, docID uuid.UUID, page domain.PageRequest) (domain.Page[domain.DocumentExport], error) {
	if _, err := s.Get(ctx, userID, docID); err != nil {
		return domain.Page[domain.DocumentExport]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.documents.ListExports(ctx, userID, docID, page, limit+1)
	if err != nil {
		return domain.Page[domain.DocumentExport]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(e domain.DocumentExport) domain.Cursor {
		return domain.Cursor{CreatedAt: e.CreatedAt, ID: e.ID}
	}), nil
}
