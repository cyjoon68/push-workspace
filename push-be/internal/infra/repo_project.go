package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const blueprintCols = `id, user_id, revision, application_id, gap_analysis_id, title, skills,
	problem, solution, tasks, completion_criteria, metrics, estimated_effort, state, created_at, updated_at`

type ProjectRepo struct{ d *DB }

func NewProjectRepo(d *DB) *ProjectRepo { return &ProjectRepo{d: d} }

func scanBlueprint(row interface{ Scan(...any) error }) (*domain.ProjectBlueprint, error) {
	b := &domain.ProjectBlueprint{}
	var skills, tasks, criteria, metrics, effort []byte
	err := row.Scan(&b.ID, &b.UserID, &b.Revision, &b.ApplicationID, &b.GapAnalysisID,
		&b.Title, &skills, &b.Problem, &b.Solution, &tasks, &criteria, &metrics,
		&effort, &b.State, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(skills, &b.Skills); err != nil {
		return nil, err
	}
	if err := unj(tasks, &b.Tasks); err != nil {
		return nil, err
	}
	if err := unj(criteria, &b.CompletionCriteria); err != nil {
		return nil, err
	}
	if err := unj(metrics, &b.Metrics); err != nil {
		return nil, err
	}
	if err := unj(effort, &b.EstimatedEffort); err != nil {
		return nil, err
	}
	if b.Skills == nil {
		b.Skills = []string{}
	}
	if b.Tasks == nil {
		b.Tasks = []domain.BlueprintTask{}
	}
	if b.Metrics == nil {
		b.Metrics = []domain.BlueprintMetric{}
	}
	if b.CompletionCriteria == nil {
		b.CompletionCriteria = []string{}
	}
	return b, nil
}

func blueprintJSON(b *domain.ProjectBlueprint) (skills, tasks, criteria, metrics, effort []byte) {
	skills, _ = jv(b.Skills)
	tasks, _ = jv(b.Tasks)
	criteria, _ = jv(b.CompletionCriteria)
	metrics, _ = jv(b.Metrics)
	effort, _ = jv(b.EstimatedEffort)
	return
}

func (r *ProjectRepo) CreateBlueprint(ctx context.Context, b *domain.ProjectBlueprint) error {
	skills, tasks, criteria, metrics, effort := blueprintJSON(b)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO project_blueprints (`+blueprintCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		b.ID, b.UserID, b.Revision, b.ApplicationID, b.GapAnalysisID, b.Title, skills,
		b.Problem, b.Solution, tasks, criteria, metrics, effort, b.State, b.CreatedAt, b.UpdatedAt)
	return err
}

func (r *ProjectRepo) GetBlueprint(ctx context.Context, userID, id uuid.UUID) (*domain.ProjectBlueprint, error) {
	b, err := scanBlueprint(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+blueprintCols+` FROM project_blueprints WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return b, nil
}

func (r *ProjectRepo) ListBlueprints(ctx context.Context, userID uuid.UUID, f domain.ProjectFilter, limit int) ([]domain.ProjectBlueprint, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if f.ApplicationID != nil {
		c.add("application_id = %s", *f.ApplicationID)
	}
	c.cursor(f.Page)
	sql, args := c.query(blueprintCols, "project_blueprints", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ProjectBlueprint{}
	for rows.Next() {
		b, err := scanBlueprint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (r *ProjectRepo) UpdateBlueprint(ctx context.Context, b *domain.ProjectBlueprint, expectedRevision int64) error {
	skills, tasks, criteria, metrics, effort := blueprintJSON(b)
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE project_blueprints SET revision = revision + 1, title = $3, skills = $4,
		 problem = $5, solution = $6, tasks = $7, completion_criteria = $8, metrics = $9,
		 estimated_effort = $10, state = $11, updated_at = $12
		 WHERE id = $1 AND user_id = $2 AND revision = $13`,
		b.ID, b.UserID, b.Title, skills, b.Problem, b.Solution, tasks, criteria,
		metrics, effort, b.State, b.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "project_blueprints", b.ID, b.UserID, expectedRevision)
	}
	return nil
}

const runCols = `id, user_id, revision, project_id, application_id, provider, working_directory,
	executable, arguments, prompt, payload_hash, state, launch_status, approval_id, device_id,
	detected_version, started_at, finished_at, failure_reason, exit_code, commit_sha,
	stdout_hash, stderr_hash, created_at, updated_at`

func scanRun(row interface{ Scan(...any) error }) (*domain.CliRun, error) {
	r := &domain.CliRun{}
	var args []byte
	err := row.Scan(&r.ID, &r.UserID, &r.Revision, &r.ProjectID, &r.ApplicationID,
		&r.Provider, &r.WorkingDirectory, &r.Executable, &args, &r.Prompt, &r.PayloadHash,
		&r.State, &r.LaunchStatus, &r.ApprovalID, &r.DeviceID, &r.DetectedVersion,
		&r.StartedAt, &r.FinishedAt, &r.FailureReason, &r.ExitCode, &r.CommitSHA,
		&r.StdoutHash, &r.StderrHash, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(args, &r.Arguments); err != nil {
		return nil, err
	}
	if r.Arguments == nil {
		r.Arguments = []string{}
	}
	return r, nil
}

func (r *ProjectRepo) CreateRun(ctx context.Context, run *domain.CliRun) error {
	args, _ := jv(run.Arguments)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO cli_runs (`+runCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`,
		run.ID, run.UserID, run.Revision, run.ProjectID, run.ApplicationID, run.Provider,
		run.WorkingDirectory, run.Executable, args, run.Prompt, run.PayloadHash, run.State,
		run.LaunchStatus, run.ApprovalID, run.DeviceID, run.DetectedVersion, run.StartedAt,
		run.FinishedAt, run.FailureReason, run.ExitCode, run.CommitSHA, run.StdoutHash,
		run.StderrHash, run.CreatedAt, run.UpdatedAt)
	return err
}

func (r *ProjectRepo) GetRun(ctx context.Context, userID, projectID, runID uuid.UUID) (*domain.CliRun, error) {
	q := `SELECT ` + runCols + ` FROM cli_runs WHERE id = $1 AND user_id = $2`
	args := []any{runID, userID}
	if projectID != uuid.Nil {
		q += ` AND project_id = $3`
		args = append(args, projectID)
	}
	run, err := scanRun(r.d.Q(ctx).QueryRow(ctx, q, args...))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return run, nil
}

func (r *ProjectRepo) ListRuns(ctx context.Context, userID, projectID uuid.UUID, page domain.PageRequest, limit int) ([]domain.CliRun, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("project_id = %s", projectID)
	c.cursor(page)
	sql, args := c.query(runCols, "cli_runs", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CliRun{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *run)
	}
	return out, rows.Err()
}

func (r *ProjectRepo) UpdateRun(ctx context.Context, run *domain.CliRun, expectedRevision int64) error {
	args, _ := jv(run.Arguments)
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE cli_runs SET revision = revision + 1, executable = $3, arguments = $4, prompt = $5,
		 payload_hash = $6, state = $7, launch_status = $8, approval_id = $9, device_id = $10,
		 detected_version = $11, started_at = $12, finished_at = $13, failure_reason = $14,
		 exit_code = $15, commit_sha = $16, stdout_hash = $17, stderr_hash = $18, updated_at = $19
		 WHERE id = $1 AND user_id = $2 AND revision = $20`,
		run.ID, run.UserID, run.Executable, args, run.Prompt, run.PayloadHash, run.State,
		run.LaunchStatus, run.ApprovalID, run.DeviceID, run.DetectedVersion, run.StartedAt,
		run.FinishedAt, run.FailureReason, run.ExitCode, run.CommitSHA, run.StdoutHash,
		run.StderrHash, run.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "cli_runs", run.ID, run.UserID, expectedRevision)
	}
	return nil
}

const pevidenceCols = `id, user_id, revision, project_id, run_id, commit_url, commit_sha,
	test_results, metrics, summary, status, verification_method, verified_at, career_evidence_id,
	created_at, updated_at`

func scanProjectEvidence(row interface{ Scan(...any) error }) (*domain.ProjectEvidence, error) {
	e := &domain.ProjectEvidence{}
	var tr, metrics []byte
	err := row.Scan(&e.ID, &e.UserID, &e.Revision, &e.ProjectID, &e.RunID, &e.CommitURL,
		&e.CommitSHA, &tr, &metrics, &e.Summary, &e.Status, &e.VerificationMethod,
		&e.VerifiedAt, &e.CareerEvidenceID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(tr, &e.TestResults); err != nil {
		return nil, err
	}
	if err := unj(metrics, &e.Metrics); err != nil {
		return nil, err
	}
	if e.Metrics == nil {
		e.Metrics = []domain.MetricEntry{}
	}
	return e, nil
}

func (r *ProjectRepo) CreateEvidence(ctx context.Context, e *domain.ProjectEvidence) error {
	tr, _ := jv(e.TestResults)
	metrics, _ := jv(e.Metrics)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO project_evidence (`+pevidenceCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		e.ID, e.UserID, e.Revision, e.ProjectID, e.RunID, e.CommitURL, e.CommitSHA,
		tr, metrics, e.Summary, e.Status, e.VerificationMethod, e.VerifiedAt,
		e.CareerEvidenceID, e.CreatedAt, e.UpdatedAt)
	return err
}

func (r *ProjectRepo) GetEvidence(ctx context.Context, userID, projectID, evidenceID uuid.UUID) (*domain.ProjectEvidence, error) {
	e, err := scanProjectEvidence(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+pevidenceCols+` FROM project_evidence WHERE id = $1 AND project_id = $2 AND user_id = $3`,
		evidenceID, projectID, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return e, nil
}

func (r *ProjectRepo) ListEvidence(ctx context.Context, userID, projectID uuid.UUID, page domain.PageRequest, limit int) ([]domain.ProjectEvidence, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("project_id = %s", projectID)
	c.cursor(page)
	sql, args := c.query(pevidenceCols, "project_evidence", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ProjectEvidence{}
	for rows.Next() {
		e, err := scanProjectEvidence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *ProjectRepo) UpdateEvidence(ctx context.Context, e *domain.ProjectEvidence, expectedRevision int64) error {
	tr, _ := jv(e.TestResults)
	metrics, _ := jv(e.Metrics)
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE project_evidence SET revision = revision + 1, status = $3, verification_method = $4,
		 verified_at = $5, career_evidence_id = $6, test_results = $7, metrics = $8, summary = $9,
		 updated_at = $10
		 WHERE id = $1 AND user_id = $2 AND revision = $11`,
		e.ID, e.UserID, e.Status, e.VerificationMethod, e.VerifiedAt, e.CareerEvidenceID,
		tr, metrics, e.Summary, e.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "project_evidence", e.ID, e.UserID, expectedRevision)
	}
	return nil
}
