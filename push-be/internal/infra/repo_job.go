package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const jobCols = `id, user_id, revision, company, title, source_kind, source_url, source_text,
	requirements, preferred, keywords, risks, deadline, language, archived, created_at, updated_at`

type JobRepo struct{ d *DB }

func NewJobRepo(d *DB) *JobRepo { return &JobRepo{d: d} }

func scanJob(row interface{ Scan(...any) error }) (*domain.JobPosting, error) {
	j := &domain.JobPosting{}
	var reqs, pref, kw, risks []byte
	err := row.Scan(&j.ID, &j.UserID, &j.Revision, &j.Company, &j.Title, &j.SourceKind,
		&j.SourceURL, &j.SourceText, &reqs, &pref, &kw, &risks, &j.Deadline, &j.Language,
		&j.Archived, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	for _, p := range []struct {
		b   []byte
		out *[]string
	}{{reqs, &j.Requirements}, {pref, &j.Preferred}, {kw, &j.Keywords}, {risks, &j.Risks}} {
		if err := unj(p.b, p.out); err != nil {
			return nil, err
		}
		if *p.out == nil {
			*p.out = []string{}
		}
	}
	return j, nil
}

func (r *JobRepo) Create(ctx context.Context, j *domain.JobPosting) error {
	reqs, _ := jv(j.Requirements)
	pref, _ := jv(j.Preferred)
	kw, _ := jv(j.Keywords)
	risks, _ := jv(j.Risks)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO jobs (`+jobCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		j.ID, j.UserID, j.Revision, j.Company, j.Title, j.SourceKind, j.SourceURL, j.SourceText,
		reqs, pref, kw, risks, j.Deadline, j.Language, j.Archived, j.CreatedAt, j.UpdatedAt)
	return err
}

func (r *JobRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.JobPosting, error) {
	j, err := scanJob(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+jobCols+` FROM jobs WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return j, nil
}

func (r *JobRepo) List(ctx context.Context, userID uuid.UUID, f domain.JobFilter, limit int) ([]domain.JobPosting, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if f.Archived != nil {
		c.add("archived = %s", *f.Archived)
	}
	if f.Query != "" {
		c.add("(company ILIKE %s OR title ILIKE %s)", "%"+f.Query+"%", "%"+f.Query+"%")
	}
	c.cursor(f.Page)
	sql, args := c.query(jobCols, "jobs", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.JobPosting{}
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func (r *JobRepo) Update(ctx context.Context, j *domain.JobPosting, expectedRevision int64) error {
	reqs, _ := jv(j.Requirements)
	pref, _ := jv(j.Preferred)
	kw, _ := jv(j.Keywords)
	risks, _ := jv(j.Risks)
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE jobs SET revision = revision + 1, company = $3, title = $4, source_kind = $5,
		 source_url = $6, source_text = $7, requirements = $8, preferred = $9, keywords = $10,
		 risks = $11, deadline = $12, language = $13, archived = $14, updated_at = $15
		 WHERE id = $1 AND user_id = $2 AND revision = $16`,
		j.ID, j.UserID, j.Company, j.Title, j.SourceKind, j.SourceURL, j.SourceText,
		reqs, pref, kw, risks, j.Deadline, j.Language, j.Archived, j.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "jobs", j.ID, j.UserID, expectedRevision)
	}
	return nil
}

const analysisCols = `id, user_id, application_id, job_id, job_revision, evidence_ids, matched,
	missing, preferred_missing, risks, fit_score, method, stale, created_at`

type GapAnalysisRepo struct{ d *DB }

func NewGapAnalysisRepo(d *DB) *GapAnalysisRepo { return &GapAnalysisRepo{d: d} }

func scanAnalysis(row interface{ Scan(...any) error }) (*domain.GapAnalysis, error) {
	g := &domain.GapAnalysis{}
	var eids, matched, missing, pref, risks []byte
	err := row.Scan(&g.ID, &g.UserID, &g.ApplicationID, &g.JobID, &g.JobRevision,
		&eids, &matched, &missing, &pref, &risks, &g.FitScore, &g.Method, &g.Stale, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(eids, &g.EvidenceIDs); err != nil {
		return nil, err
	}
	if err := unj(matched, &g.Matched); err != nil {
		return nil, err
	}
	for _, p := range []struct {
		b   []byte
		out *[]string
	}{{missing, &g.Missing}, {pref, &g.PreferredMissing}, {risks, &g.Risks}} {
		if err := unj(p.b, p.out); err != nil {
			return nil, err
		}
		if *p.out == nil {
			*p.out = []string{}
		}
	}
	if g.EvidenceIDs == nil {
		g.EvidenceIDs = []uuid.UUID{}
	}
	if g.Matched == nil {
		g.Matched = []domain.RequirementMatch{}
	}
	return g, nil
}

func (r *GapAnalysisRepo) Create(ctx context.Context, g *domain.GapAnalysis) error {
	eids, _ := jv(g.EvidenceIDs)
	matched, _ := jv(g.Matched)
	missing, _ := jv(g.Missing)
	pref, _ := jv(g.PreferredMissing)
	risks, _ := jv(g.Risks)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO gap_analyses (`+analysisCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		g.ID, g.UserID, g.ApplicationID, g.JobID, g.JobRevision, eids, matched,
		missing, pref, risks, g.FitScore, g.Method, g.Stale, g.CreatedAt)
	return err
}

func (r *GapAnalysisRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.GapAnalysis, error) {
	g, err := scanAnalysis(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+analysisCols+` FROM gap_analyses WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return g, nil
}

func (r *GapAnalysisRepo) ListByJob(ctx context.Context, userID, jobID uuid.UUID, page domain.PageRequest, limit int) ([]domain.GapAnalysis, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("job_id = %s", jobID)
	c.cursor(page)
	sql, args := c.query(analysisCols, "gap_analyses", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GapAnalysis{}
	for rows.Next() {
		g, err := scanAnalysis(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, rows.Err()
}

func (r *GapAnalysisRepo) MarkStaleForJob(ctx context.Context, userID, jobID uuid.UUID, exceptID uuid.UUID) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE gap_analyses SET stale = true WHERE user_id = $1 AND job_id = $2 AND id <> $3`,
		userID, jobID, exceptID)
	return err
}

func (r *GapAnalysisRepo) CountByApplication(ctx context.Context, userID, applicationID uuid.UUID) (int, error) {
	var n int
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM gap_analyses WHERE user_id = $1 AND application_id = $2`,
		userID, applicationID).Scan(&n)
	return n, err
}
