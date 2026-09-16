package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type ProjectService struct {
	projects     domain.ProjectStore
	applications domain.ApplicationStore
	analyses     domain.GapAnalysisStore
	approvals    *ApprovalService
	ops          domain.OperationStore
	uow          domain.UnitOfWork
	ai           *AIGate
}

func NewProjectService(projects domain.ProjectStore, applications domain.ApplicationStore, analyses domain.GapAnalysisStore, approvals *ApprovalService, ops domain.OperationStore, uow domain.UnitOfWork, ai *AIGate) *ProjectService {
	return &ProjectService{projects: projects, applications: applications, analyses: analyses, approvals: approvals, ops: ops, uow: uow, ai: ai}
}

func (s *ProjectService) ListBlueprints(ctx context.Context, userID uuid.UUID, f domain.ProjectFilter) (domain.Page[domain.ProjectBlueprint], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.projects.ListBlueprints(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.ProjectBlueprint]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(b domain.ProjectBlueprint) domain.Cursor {
		return domain.Cursor{CreatedAt: b.CreatedAt, ID: b.ID}
	}), nil
}

func (s *ProjectService) GetBlueprint(ctx context.Context, userID, id uuid.UUID) (*domain.ProjectBlueprint, error) {
	b, err := s.projects.GetBlueprint(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return b, nil
}

func (s *ProjectService) GenerateBlueprints(ctx context.Context, userID, applicationID, gapAnalysisID uuid.UUID, ai *domain.AiOptions) (*domain.Operation, error) {
	if err := s.ai.Check(ctx, userID, ai); err != nil {
		return nil, err
	}
	var op *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		app, err := s.applications.Get(ctx, userID, applicationID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		analysis, err := s.analyses.Get(ctx, userID, gapAnalysisID)
		if err != nil {
			return domain.NotFound()
		}
		if analysis.ApplicationID != applicationID {
			return domain.ValidationField("gapAnalysisId", "analysis belongs to another application")
		}
		now := time.Now().UTC()
		blueprints := buildBlueprints(userID, applicationID, gapAnalysisID, analysis, now)
		for _, b := range blueprints {
			if err := s.projects.CreateBlueprint(ctx, b); err != nil {
				return err
			}
		}
		res, _ := json.Marshal(blueprints)
		op = &domain.Operation{
			ID: uuid.New(), UserID: userID, Type: domain.OpProjectBlueprints,
			ApplicationID: &app.ID, Status: domain.OpSucceeded,
			Result:    &domain.OperationResult{Kind: string(domain.OpProjectBlueprints), Value: res},
			CreatedAt: now, UpdatedAt: now,
		}
		return s.ops.Create(ctx, op)
	})
	if err != nil {
		return nil, err
	}
	return op, nil
}

func buildBlueprints(userID, applicationID, gapAnalysisID uuid.UUID, a *domain.GapAnalysis, now time.Time) []*domain.ProjectBlueprint {
	gaps := append(append([]string{}, a.Missing...), a.PreferredMissing...)
	if len(gaps) == 0 {
		gaps = []string{"general engineering practice"}
	}
	out := []*domain.ProjectBlueprint{}
	for i := 0; i < 4; i++ {
		skill := gaps[i%len(gaps)]
		out = append(out, &domain.ProjectBlueprint{
			ID: uuid.New(), UserID: userID, Revision: 1,
			ApplicationID: applicationID, GapAnalysisID: gapAnalysisID,
			Title:    fmt.Sprintf("Gap project %d: %s", i+1, skill),
			Skills:   []string{skill},
			Problem:  "Missing or unproven requirement: " + skill,
			Solution: "Build a small project that exercises " + skill,
			Tasks: []domain.BlueprintTask{{
				ID: "task-1", Title: "Implement core feature using " + skill,
				Description: "Create a minimal but working implementation",
				Acceptance:  []string{"runs locally", "tests pass"},
			}},
			CompletionCriteria: []string{"committed to a remote repository", "tests pass in CI"},
			Metrics: []domain.BlueprintMetric{{
				Name: "test_coverage", Unit: "percent", Measurement: "coverage report", Target: nil,
			}},
			EstimatedEffort: domain.EffortEstimate{MinHours: 4, MaxHours: 16},
			State:           domain.BlueprintDraft, CreatedAt: now, UpdatedAt: now,
		})
	}
	return out
}

type ManifestFile struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	SHA256   string `json:"sha256"`
}

type ProjectManifest struct {
	ProjectID         uuid.UUID      `json:"projectId"`
	BlueprintRevision int64          `json:"blueprintRevision"`
	Files             []ManifestFile `json:"files"`
}

func buildManifest(b *domain.ProjectBlueprint) (*ProjectManifest, *domain.Error) {
	files := []ManifestFile{}
	readme := "# " + b.Title + "\n\n## Problem\n" + b.Problem + "\n\n## Solution\n" + b.Solution + "\n"
	tasks := ""
	for _, t := range b.Tasks {
		tasks += "## " + t.Title + "\n" + t.Description + "\n\nAcceptance:\n"
		for _, a := range t.Acceptance {
			tasks += "- " + a + "\n"
		}
		tasks += "\n"
	}
	for _, f := range [][2]string{
		{"README.md", readme},
		{"docs/tasks.md", tasks},
	} {
		if !validManifestPath(f[0]) {
			return nil, domain.Internal()
		}
		files = append(files, ManifestFile{Path: f[0], Encoding: "utf8", Content: f[1], SHA256: domain.HashBytes([]byte(f[1]))})
	}
	return &ProjectManifest{ProjectID: b.ID, BlueprintRevision: b.Revision, Files: files}, nil
}

func validManifestPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\") {
		return false
	}
	clean := path.Clean(p)
	if clean != p || strings.HasPrefix(clean, "..") || strings.Contains(clean, "/../") {
		return false
	}
	return true
}

func (s *ProjectService) Select(ctx context.Context, userID, id uuid.UUID, expectedRevision int64) (*domain.ProjectBlueprint, *ProjectManifest, error) {
	var out *domain.ProjectBlueprint
	var manifest *ProjectManifest
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		b, err := s.projects.GetBlueprint(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if b.Revision != expectedRevision {
			return domain.RevisionConflict(b.Revision)
		}
		if b.State != domain.BlueprintDraft && b.State != domain.BlueprintSelected {
			return domain.InvalidTransition("blueprint cannot be selected from " + string(b.State))
		}
		b.State = domain.BlueprintSelected
		b.UpdatedAt = time.Now().UTC()
		if err := s.projects.UpdateBlueprint(ctx, b, expectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		b.Revision = expectedRevision + 1
		m, err := buildManifest(b)
		if err != nil {
			return err
		}
		m.BlueprintRevision = b.Revision
		manifest = m
		out = b
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return out, manifest, nil
}

var cliExecutables = map[domain.CliProvider]string{
	domain.CliCodex:      "codex",
	domain.CliClaudeCode: "claude",
	domain.CliGrokBuild:  "grok",
}

func runPayloadHash(provider domain.CliProvider, dir, executable string, args []string, prompt string) string {
	return domain.HashJSON(map[string]any{
		"provider": provider, "workingDirectory": dir,
		"executable": executable, "arguments": args, "prompt": prompt,
	})
}

func (s *ProjectService) ListRuns(ctx context.Context, userID, projectID uuid.UUID, page domain.PageRequest) (domain.Page[domain.CliRun], error) {
	if _, err := s.GetBlueprint(ctx, userID, projectID); err != nil {
		return domain.Page[domain.CliRun]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.projects.ListRuns(ctx, userID, projectID, page, limit+1)
	if err != nil {
		return domain.Page[domain.CliRun]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(r domain.CliRun) domain.Cursor {
		return domain.Cursor{CreatedAt: r.CreatedAt, ID: r.ID}
	}), nil
}

func (s *ProjectService) GetRun(ctx context.Context, userID, projectID, runID uuid.UUID) (*domain.CliRun, error) {
	r, err := s.projects.GetRun(ctx, userID, projectID, runID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return r, nil
}

func (s *ProjectService) CreateRun(ctx context.Context, userID, projectID uuid.UUID, provider domain.CliProvider, workingDirectory, prompt string) (*domain.CliRun, error) {
	if !domain.ValidCliProvider(provider) {
		return nil, domain.ValidationField("provider", "unsupported provider")
	}
	if workingDirectory == "" || !path.IsAbs(workingDirectory) {
		return nil, domain.ValidationField("workingDirectory", "absolute path required")
	}
	if prompt == "" {
		return nil, domain.ValidationField("prompt", "required")
	}
	var out *domain.CliRun
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		b, err := s.projects.GetBlueprint(ctx, userID, projectID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if b.State != domain.BlueprintSelected && b.State != domain.BlueprintInProgress {
			return domain.InvalidTransition("blueprint must be selected before creating a run")
		}
		executable := cliExecutables[provider]
		args := []string{}
		now := time.Now().UTC()
		r := &domain.CliRun{
			ID: uuid.New(), UserID: userID, Revision: 1,
			ProjectID: projectID, ApplicationID: b.ApplicationID,
			Provider: provider, WorkingDirectory: workingDirectory,
			Executable: executable, Arguments: args, Prompt: prompt,
			PayloadHash: runPayloadHash(provider, workingDirectory, executable, args, prompt),
			State:       domain.CliRunApprovalRequired, LaunchStatus: domain.LaunchNotClaimed,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.projects.CreateRun(ctx, r); err != nil {
			return err
		}
		if b.State == domain.BlueprintSelected {
			b.State = domain.BlueprintInProgress
			b.UpdatedAt = now
			if err := s.projects.UpdateBlueprint(ctx, b, b.Revision); err != nil {
				return mapRevisionErr(err)
			}
		}
		out = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type StartRunInput struct {
	ExpectedRevision int64
	ApprovalID       uuid.UUID
	DetectedVersion  string
	DeviceID         string
}

func (s *ProjectService) StartRun(ctx context.Context, userID, projectID, runID uuid.UUID, in StartRunInput) (*domain.CliRun, error) {
	if in.DeviceID == "" {
		return nil, domain.ValidationField("deviceId", "required")
	}
	if in.DetectedVersion == "" {
		return nil, domain.ValidationField("detectedVersion", "required")
	}
	var out *domain.CliRun
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		r, err := s.projects.GetRun(ctx, userID, projectID, runID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if r.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(r.Revision)
		}
		if r.State != domain.CliRunApprovalRequired {
			return domain.InvalidTransition("run cannot start from " + string(r.State))
		}
		ap, err := s.approvals.Consume(ctx, userID, in.ApprovalID, domain.ApprovalCliExecute, r.ApplicationID, r.ID, r.PayloadHash)
		if err != nil {
			return err
		}
		if ap.TargetRevision != nil && *ap.TargetRevision != r.Revision {
			return domain.ApprovalStale("approval was issued for a different run revision")
		}
		now := time.Now().UTC()
		r.State = domain.CliRunRunning
		r.ApprovalID = &in.ApprovalID
		r.DeviceID = &in.DeviceID
		r.DetectedVersion = &in.DetectedVersion
		r.StartedAt = &now
		r.UpdatedAt = now
		if err := s.projects.UpdateRun(ctx, r, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		r.Revision = in.ExpectedRevision + 1
		out = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type LaunchInput struct {
	ExpectedRevision int64
	DeviceID         string
	PayloadHash      string
	LaunchStatus     domain.LaunchStatus
	Process          *domain.ProcessInfo
}

func (s *ProjectService) ReportLaunch(ctx context.Context, userID, projectID, runID uuid.UUID, in LaunchInput) (*domain.CliRun, error) {
	if !domain.ValidReportedLaunchStatus(in.LaunchStatus) {
		return nil, domain.ValidationField("launchStatus", "must be CLAIMED, STARTED or UNKNOWN")
	}
	var out *domain.CliRun
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		r, err := s.projects.GetRun(ctx, userID, projectID, runID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if r.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(r.Revision)
		}
		if r.State != domain.CliRunRunning {
			return domain.InvalidTransition("run is not running")
		}
		if r.DeviceID == nil || *r.DeviceID != in.DeviceID {
			return domain.InvalidTransition("run is bound to a different device")
		}
		if in.PayloadHash != r.PayloadHash {
			return domain.ApprovalStale("payload hash does not match the approved run")
		}
		if in.LaunchStatus == domain.LaunchStarted && in.Process == nil {
			return domain.ValidationField("process", "process info required for STARTED")
		}
		if !domain.CanTransitionLaunch(r.LaunchStatus, in.LaunchStatus) {
			return domain.InvalidTransition("launchStatus cannot move from " + string(r.LaunchStatus) + " to " + string(in.LaunchStatus))
		}
		r.LaunchStatus = in.LaunchStatus
		r.UpdatedAt = time.Now().UTC()
		if err := s.projects.UpdateRun(ctx, r, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		r.Revision = in.ExpectedRevision + 1
		out = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type RecoverInput struct {
	ExpectedRevision int64
	DeviceID         string
	Decision         string
	Process          *domain.ProcessInfo
	FailureReason    *string
}

func (s *ProjectService) Recover(ctx context.Context, userID, projectID, runID uuid.UUID, in RecoverInput) (*domain.CliRun, error) {
	if in.Decision != "REATTACH" && in.Decision != "MARK_FAILED" {
		return nil, domain.ValidationField("decision", "must be REATTACH or MARK_FAILED")
	}
	var out *domain.CliRun
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		r, err := s.projects.GetRun(ctx, userID, projectID, runID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if r.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(r.Revision)
		}
		if r.LaunchStatus != domain.LaunchUnknown {
			return domain.InvalidTransition("recover is only allowed from UNKNOWN launch status")
		}
		if r.DeviceID == nil || *r.DeviceID != in.DeviceID {
			return domain.InvalidTransition("run is bound to a different device")
		}
		now := time.Now().UTC()
		if in.Decision == "REATTACH" {
			if in.Process == nil || in.Process.PID <= 0 {
				return domain.ValidationField("process", "process info required for REATTACH")
			}
			r.LaunchStatus = domain.LaunchStarted
		} else {
			if in.FailureReason == nil || *in.FailureReason == "" {
				return domain.ValidationField("failureReason", "required for MARK_FAILED")
			}
			r.LaunchStatus = domain.LaunchFinished
			r.State = domain.CliRunFailed
			r.FailureReason = in.FailureReason
			r.FinishedAt = &now
		}
		r.UpdatedAt = now
		if err := s.projects.UpdateRun(ctx, r, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		r.Revision = in.ExpectedRevision + 1
		out = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type RunResultInput struct {
	ExpectedRevision int64
	ExitCode         int
	CommitSHA        *string
	StdoutHash       string
	StderrHash       string
}

func (s *ProjectService) ReportResult(ctx context.Context, userID, projectID, runID uuid.UUID, in RunResultInput) (*domain.CliRun, error) {
	var out *domain.CliRun
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		r, err := s.projects.GetRun(ctx, userID, projectID, runID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if r.Revision != in.ExpectedRevision {
			return domain.RevisionConflict(r.Revision)
		}
		if r.State != domain.CliRunRunning || r.LaunchStatus != domain.LaunchStarted {
			return domain.InvalidTransition("result requires a RUNNING run with STARTED launch status")
		}
		now := time.Now().UTC()
		r.LaunchStatus = domain.LaunchFinished
		r.ExitCode = &in.ExitCode
		r.CommitSHA = in.CommitSHA
		r.StdoutHash = &in.StdoutHash
		r.StderrHash = &in.StderrHash
		r.FinishedAt = &now
		if in.ExitCode == 0 {
			r.State = domain.CliRunVerifying
		} else {
			r.State = domain.CliRunFailed
			reason := fmt.Sprintf("exit code %d", in.ExitCode)
			r.FailureReason = &reason
		}
		r.UpdatedAt = now
		if err := s.projects.UpdateRun(ctx, r, in.ExpectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		r.Revision = in.ExpectedRevision + 1
		out = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type CreateProjectEvidenceInput struct {
	RunID       uuid.UUID
	CommitURL   string
	TestCommand string
	TestOutput  string
	ExitCode    int
	Metrics     []domain.MetricEntry
	Summary     string
}

func (s *ProjectService) CreateEvidence(ctx context.Context, userID, projectID uuid.UUID, in CreateProjectEvidenceInput) (*domain.ProjectEvidence, error) {
	if err := domain.ValidateHTTPSURL(in.CommitURL); err != nil {
		return nil, domain.ValidationField("commitUrl", "must be an http(s) URL")
	}
	var out *domain.ProjectEvidence
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		b, err := s.projects.GetBlueprint(ctx, userID, projectID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		r, err := s.projects.GetRun(ctx, userID, projectID, in.RunID)
		if err != nil {
			return domain.ValidationField("runId", "run not found")
		}
		if r.State != domain.CliRunVerifying && r.State != domain.CliRunVerified {
			return domain.InvalidTransition("run must be VERIFYING before evidence is submitted")
		}
		now := time.Now().UTC()
		e := &domain.ProjectEvidence{
			ID: uuid.New(), UserID: userID, Revision: 1,
			ProjectID: projectID, RunID: in.RunID, CommitURL: in.CommitURL,
			CommitSHA: r.CommitSHA,
			TestResults: domain.TestResults{
				TestCommand: in.TestCommand, TestOutput: in.TestOutput, ExitCode: in.ExitCode,
			},
			Metrics: in.Metrics, Summary: in.Summary,
			Status: domain.ProjectEvidencePending, CreatedAt: now, UpdatedAt: now,
		}
		if e.Metrics == nil {
			e.Metrics = []domain.MetricEntry{}
		}
		if err := s.projects.CreateEvidence(ctx, e); err != nil {
			return err
		}
		_ = b
		out = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ProjectService) ListEvidence(ctx context.Context, userID, projectID uuid.UUID, page domain.PageRequest) (domain.Page[domain.ProjectEvidence], error) {
	if _, err := s.GetBlueprint(ctx, userID, projectID); err != nil {
		return domain.Page[domain.ProjectEvidence]{}, err
	}
	limit := page.EffectiveLimit()
	items, err := s.projects.ListEvidence(ctx, userID, projectID, page, limit+1)
	if err != nil {
		return domain.Page[domain.ProjectEvidence]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(e domain.ProjectEvidence) domain.Cursor {
		return domain.Cursor{CreatedAt: e.CreatedAt, ID: e.ID}
	}), nil
}

func (s *ProjectService) VerifyEvidence(ctx context.Context, userID, projectID, evidenceID uuid.UUID, expectedRevision int64) (*domain.Operation, error) {
	var op *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		e, err := s.projects.GetEvidence(ctx, userID, projectID, evidenceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if e.Revision != expectedRevision {
			return domain.RevisionConflict(e.Revision)
		}
		if e.Status == domain.ProjectEvidenceVerified {
			return domain.InvalidTransition("evidence already verified")
		}
		now := time.Now().UTC()
		method := "UNAVAILABLE"
		e.VerificationMethod = &method
		e.UpdatedAt = now
		if err := s.projects.UpdateEvidence(ctx, e, expectedRevision); err != nil {
			return mapRevisionErr(err)
		}
		e.Revision = expectedRevision + 1
		res, _ := json.Marshal(e)
		op = &domain.Operation{
			ID: uuid.New(), UserID: userID, Type: domain.OpProjectVerify,
			Status:    domain.OpSucceeded,
			Result:    &domain.OperationResult{Kind: string(domain.OpProjectVerify), Value: res},
			CreatedAt: now, UpdatedAt: now,
		}
		b, err := s.projects.GetBlueprint(ctx, userID, projectID)
		if err == nil {
			op.ApplicationID = &b.ApplicationID
		}
		return s.ops.Create(ctx, op)
	})
	if err != nil {
		return nil, err
	}
	return op, nil
}
