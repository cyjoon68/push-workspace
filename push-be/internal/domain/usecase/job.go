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

type JobService struct {
	jobs         domain.JobStore
	analyses     domain.GapAnalysisStore
	applications domain.ApplicationStore
	evidence     domain.EvidenceStore
	approvals    *ApprovalService
	ops          domain.OperationStore
	uow          domain.UnitOfWork
	ai           *AIGate
}

type AIGate struct {
	ManagedConfigured bool
	ManagedKey        string
	ByokEnabled       bool
	Keys              domain.AiKeyStore
	Cipher            domain.KeyCipher
	Chat              domain.ChatCompleter
}

func (g *AIGate) Check(ctx context.Context, userID uuid.UUID, ai *domain.AiOptions) *domain.Error {
	return domain.CheckAICredentials(ai, g.ManagedConfigured, func(provider string) bool {
		if !g.ByokEnabled || g.Keys == nil {
			return false
		}
		_, err := g.Keys.Get(ctx, userID, provider)
		return err == nil
	})
}

func (g *AIGate) resolveKey(ctx context.Context, userID uuid.UUID, ai *domain.AiOptions) (string, *domain.Error) {
	if ai.CredentialMode == "MANAGED" {
		return g.ManagedKey, nil
	}
	k, err := g.Keys.Get(ctx, userID, ai.Provider)
	if err != nil {
		return "", domain.IntegrationRequired("no BYOK key configured for " + ai.Provider)
	}
	if g.Cipher == nil {
		return "", domain.NotConfigured("BYOK encryption is not configured")
	}
	key, err := g.Cipher.Decrypt(k.Ciphertext, k.Nonce, userID, ai.Provider)
	if err != nil {
		return "", domain.IntegrationRequired("BYOK key could not be decrypted")
	}
	return key, nil
}

func (g *AIGate) Complete(ctx context.Context, userID uuid.UUID, ai *domain.AiOptions, system, user string) (*domain.AICompletion, *domain.Error) {
	if err := g.Check(ctx, userID, ai); err != nil {
		return nil, err
	}
	if ai.Provider != "OPENAI" || g.Chat == nil {
		return nil, domain.NotConfigured("AI provider " + ai.Provider + " is not supported")
	}
	key, derr := g.resolveKey(ctx, userID, ai)
	if derr != nil {
		return nil, derr
	}
	c, err := g.Chat.Chat(ctx, key, ai.Model, system, user)
	if err != nil {
		return nil, domain.ProviderError("AI provider request failed")
	}
	return c, nil
}

func NewJobService(jobs domain.JobStore, analyses domain.GapAnalysisStore, applications domain.ApplicationStore, evidence domain.EvidenceStore, approvals *ApprovalService, ops domain.OperationStore, uow domain.UnitOfWork, ai *AIGate) *JobService {
	return &JobService{jobs: jobs, analyses: analyses, applications: applications, evidence: evidence, approvals: approvals, ops: ops, uow: uow, ai: ai}
}

type CreateJobInput struct {
	Company      string
	Title        string
	SourceKind   domain.JobSourceKind
	SourceURL    *string
	SourceText   string
	Requirements []string
	Preferred    []string
	Deadline     *time.Time
	Language     string
}

func (s *JobService) List(ctx context.Context, userID uuid.UUID, f domain.JobFilter) (domain.Page[domain.JobPosting], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.jobs.List(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.JobPosting]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(j domain.JobPosting) domain.Cursor {
		return domain.Cursor{CreatedAt: j.CreatedAt, ID: j.ID}
	}), nil
}

func (s *JobService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.JobPosting, error) {
	j, err := s.jobs.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return j, nil
}

func (s *JobService) Create(ctx context.Context, userID uuid.UUID, in CreateJobInput) (*domain.JobPosting, error) {
	if err := validateJobInput(in.Company, in.Title, in.SourceKind, in.SourceURL, in.SourceText); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	j := &domain.JobPosting{
		ID: uuid.New(), UserID: userID, Revision: 1,
		Company: in.Company, Title: in.Title, SourceKind: in.SourceKind,
		SourceURL: in.SourceURL, SourceText: in.SourceText,
		Requirements: nonNil(in.Requirements), Preferred: nonNil(in.Preferred),
		Keywords: []string{}, Risks: []string{},
		Deadline: in.Deadline, Language: in.Language,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.jobs.Create(ctx, j); err != nil {
		return nil, domain.Internal()
	}
	return j, nil
}

func validateJobInput(company, title string, kind domain.JobSourceKind, sourceURL *string, sourceText string) *domain.Error {
	if len(company) < 1 || len(company) > 200 {
		return domain.ValidationField("company", "company must be 1-200 characters")
	}
	if len(title) < 1 || len(title) > 300 {
		return domain.ValidationField("title", "title must be 1-300 characters")
	}
	if !domain.ValidJobSourceKind(kind) {
		return domain.ValidationField("sourceKind", "must be URL, TEXT or DOM")
	}
	if kind != domain.JobSourceText {
		if sourceURL == nil || *sourceURL == "" {
			return domain.ValidationField("sourceUrl", "required for URL/DOM sources")
		}
	}
	if sourceURL != nil && *sourceURL != "" {
		if err := domain.ValidateHTTPSURL(*sourceURL); err != nil {
			return domain.ValidationField("sourceUrl", "must be an http(s) URL without credentials")
		}
	}
	if n := len(sourceText); n < 1 || n > 200000 {
		return domain.ValidationField("sourceText", "collected source text is required (1-200000 characters)")
	}
	return nil
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

type PatchJobInput struct {
	ExpectedRevision int64
	Company          *string
	Title            *string
	Requirements     *[]string
	Preferred        *[]string
	Deadline         *time.Time
	ClearDeadline    bool
}

func (s *JobService) Patch(ctx context.Context, userID, id uuid.UUID, in PatchJobInput) (*domain.JobPosting, error) {
	var out *domain.JobPosting
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		j, err := s.jobs.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if in.Company != nil {
			if len(*in.Company) < 1 || len(*in.Company) > 200 {
				return domain.ValidationField("company", "company must be 1-200 characters")
			}
			j.Company = *in.Company
		}
		if in.Title != nil {
			if len(*in.Title) < 1 || len(*in.Title) > 300 {
				return domain.ValidationField("title", "title must be 1-300 characters")
			}
			j.Title = *in.Title
		}
		if in.Requirements != nil {
			j.Requirements = nonNil(*in.Requirements)
		}
		if in.Preferred != nil {
			j.Preferred = nonNil(*in.Preferred)
		}
		if in.ClearDeadline {
			j.Deadline = nil
		} else if in.Deadline != nil {
			j.Deadline = in.Deadline
		}
		j.UpdatedAt = time.Now().UTC()
		if err := s.jobs.Update(ctx, j, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		j.Revision = in.ExpectedRevision + 1
		out = j
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type AnalyzeInput struct {
	ApplicationID    uuid.UUID
	ExpectedRevision int64
	EvidenceIDs      []uuid.UUID
	AI               *domain.AiOptions
}

func (s *JobService) Analyze(ctx context.Context, userID, jobID uuid.UUID, in AnalyzeInput) (*domain.Operation, error) {
	if err := s.ai.Check(ctx, userID, in.AI); err != nil {
		return nil, err
	}
	var op *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		j, err := s.jobs.Get(ctx, userID, jobID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if j.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(j.Revision)
		}
		app, err := s.applications.Get(ctx, userID, in.ApplicationID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if app.JobID != jobID {
			return domain.ValidationField("applicationId", "application does not belong to this job")
		}
		pool := []*domain.CareerEvidence{}
		for _, eid := range in.EvidenceIDs {
			e, err := s.evidence.Get(ctx, userID, eid)
			if err != nil {
				return domain.ValidationField("evidenceIds", "evidence "+eid.String()+" not found")
			}
			if e.Archived {
				return domain.ValidationField("evidenceIds", "evidence "+eid.String()+" is archived")
			}
			if err := s.approvals.RequireEvidenceUse(ctx, userID, app.ID, eid); err != nil {
				return err
			}
			pool = append(pool, e)
		}
		method := domain.MethodRuleBased
		if in.AI != nil {
			method = domain.MethodAIAssisted
		}
		analysis := runGapAnalysis(j, app.ID, pool, method)
		if err := s.analyses.MarkStaleForJob(ctx, userID, jobID, uuid.Nil); err != nil {
			return err
		}
		if err := s.analyses.Create(ctx, analysis); err != nil {
			return err
		}
		now := time.Now().UTC()
		v, _ := json.Marshal(analysis)
		op = &domain.Operation{
			ID: uuid.New(), UserID: userID, Type: domain.OpJobAnalysis,
			ApplicationID: &app.ID, Status: domain.OpSucceeded,
			Result:    &domain.OperationResult{Kind: string(domain.OpJobAnalysis), Value: v},
			CreatedAt: now, UpdatedAt: now,
		}
		return s.ops.Create(ctx, op)
	})
	if err != nil {
		return nil, err
	}
	return op, nil
}

func runGapAnalysis(j *domain.JobPosting, appID uuid.UUID, pool []*domain.CareerEvidence, method domain.AnalysisMethod) *domain.GapAnalysis {
	matched := []domain.RequirementMatch{}
	missing := []string{}
	for _, req := range j.Requirements {
		ids := matchEvidence(req, pool)
		if len(ids) > 0 {
			matched = append(matched, domain.RequirementMatch{Requirement: req, EvidenceIDs: ids})
		} else {
			missing = append(missing, req)
		}
	}
	preferredMissing := []string{}
	for _, pref := range j.Preferred {
		if len(matchEvidence(pref, pool)) == 0 {
			preferredMissing = append(preferredMissing, pref)
		}
	}
	var fit *int
	if len(j.Requirements) > 0 {
		score := len(matched) * 100 / len(j.Requirements)
		fit = &score
	}
	eids := []uuid.UUID{}
	for _, e := range pool {
		eids = append(eids, e.ID)
	}
	return &domain.GapAnalysis{
		ID: uuid.New(), UserID: j.UserID, ApplicationID: appID, JobID: j.ID,
		JobRevision: j.Revision, EvidenceIDs: eids,
		Matched: matched, Missing: missing, PreferredMissing: preferredMissing,
		Risks: nonNil(j.Risks), FitScore: fit, Method: method,
		CreatedAt: time.Now().UTC(),
	}
}

func matchEvidence(requirement string, pool []*domain.CareerEvidence) []string {
	ids := []string{}
	req := strings.ToLower(requirement)
	words := significantWords(req)
	for _, e := range pool {
		hit := false
		for _, sk := range e.Skills {
			if sk != "" && strings.Contains(req, strings.ToLower(sk)) {
				hit = true
				break
			}
		}
		if !hit {
			src := strings.ToLower(e.SourceText)
			for _, w := range words {
				if strings.Contains(src, w) {
					hit = true
					break
				}
			}
		}
		if hit {
			ids = append(ids, e.ID.String())
		}
	}
	return ids
}

func significantWords(s string) []string {
	out := []string{}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '+' && r != '#' && !(r >= '가' && r <= '힣')
	}) {
		if len(w) >= 3 {
			out = append(out, w)
		}
	}
	return out
}

func (s *JobService) ListAnalyses(ctx context.Context, userID, jobID uuid.UUID, page domain.PageRequest) (domain.Page[domain.GapAnalysis], error) {
	if _, err := s.Get(ctx, userID, jobID); err != nil {
		return domain.Page[domain.GapAnalysis]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.analyses.ListByJob(ctx, userID, jobID, page, limit+1)
	if err != nil {
		return domain.Page[domain.GapAnalysis]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(g domain.GapAnalysis) domain.Cursor {
		return domain.Cursor{CreatedAt: g.CreatedAt, ID: g.ID}
	}), nil
}
