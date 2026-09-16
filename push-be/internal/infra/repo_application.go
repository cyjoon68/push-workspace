package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const appCols = `id, user_id, revision, job_id, company, title, stage, notes,
	applied_at, next_action_at, imported, created_at, updated_at`

type ApplicationRepo struct{ d *DB }

func NewApplicationRepo(d *DB) *ApplicationRepo { return &ApplicationRepo{d: d} }

func scanApplication(row interface{ Scan(...any) error }) (*domain.Application, error) {
	a := &domain.Application{}
	err := row.Scan(&a.ID, &a.UserID, &a.Revision, &a.JobID, &a.Company, &a.Title,
		&a.Stage, &a.Notes, &a.AppliedAt, &a.NextActionAt, &a.Imported, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *ApplicationRepo) Create(ctx context.Context, a *domain.Application) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO applications (`+appCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		a.ID, a.UserID, a.Revision, a.JobID, a.Company, a.Title, a.Stage, a.Notes,
		a.AppliedAt, a.NextActionAt, a.Imported, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *ApplicationRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Application, error) {
	a, err := scanApplication(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+appCols+` FROM applications WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return a, nil
}

func (r *ApplicationRepo) List(ctx context.Context, userID uuid.UUID, f domain.ApplicationFilter, limit int) ([]domain.Application, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if f.Stage != nil {
		c.add("stage = %s", *f.Stage)
	}
	if f.Query != "" {
		c.add("(company ILIKE %s OR title ILIKE %s)", "%"+f.Query+"%", "%"+f.Query+"%")
	}
	c.cursor(f.Page)
	sql, args := c.query(appCols, "applications", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Application{}
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *ApplicationRepo) Update(ctx context.Context, a *domain.Application, expectedRevision int64) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE applications SET revision = revision + 1, stage = $3, notes = $4,
		 applied_at = $5, next_action_at = $6, imported = $7, updated_at = $8
		 WHERE id = $1 AND user_id = $2 AND revision = $9`,
		a.ID, a.UserID, a.Stage, a.Notes, a.AppliedAt, a.NextActionAt, a.Imported,
		a.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "applications", a.ID, a.UserID, expectedRevision)
	}
	return nil
}

func (r *ApplicationRepo) AddEvent(ctx context.Context, e *domain.ApplicationEvent) error {
	payload, _ := jv(e.Payload)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO application_events (id, user_id, application_id, type, payload, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		e.ID, e.UserID, e.ApplicationID, e.Type, payload, e.CreatedAt)
	return err
}

func (r *ApplicationRepo) ListEvents(ctx context.Context, userID, applicationID uuid.UUID, page domain.PageRequest, limit int) ([]domain.ApplicationEvent, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("application_id = %s", applicationID)
	c.cursor(page)
	sql, args := c.query(`id, user_id, application_id, type, payload, created_at`, "application_events", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ApplicationEvent{}
	for rows.Next() {
		e := domain.ApplicationEvent{}
		var payload []byte
		if err := rows.Scan(&e.ID, &e.UserID, &e.ApplicationID, &e.Type, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		if err := unj(payload, &e.Payload); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type SubmissionRepo struct{ d *DB }

func NewSubmissionRepo(d *DB) *SubmissionRepo { return &SubmissionRepo{d: d} }

func (r *SubmissionRepo) CreateDraft(ctx context.Context, d *domain.SubmissionDraft) error {
	vids, _ := jv(d.DocumentVersionIDs)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO submission_drafts (id, user_id, application_id, application_revision, mode, adapter,
		 document_version_ids, confirmed_submitted, payload_hash, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		d.ID, d.UserID, d.ApplicationID, d.ApplicationRevision, d.Mode, d.Adapter,
		vids, d.ConfirmedSubmitted, d.PayloadHash, d.CreatedAt)
	return err
}

func (r *SubmissionRepo) GetDraft(ctx context.Context, userID, id uuid.UUID) (*domain.SubmissionDraft, error) {
	d := &domain.SubmissionDraft{}
	var vids []byte
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT id, user_id, application_id, application_revision, mode, adapter,
		 document_version_ids, confirmed_submitted, payload_hash, created_at
		 FROM submission_drafts WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&d.ID, &d.UserID, &d.ApplicationID, &d.ApplicationRevision, &d.Mode, &d.Adapter,
			&vids, &d.ConfirmedSubmitted, &d.PayloadHash, &d.CreatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	if err := unj(vids, &d.DocumentVersionIDs); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *SubmissionRepo) Create(ctx context.Context, s *domain.Submission) error {
	vids, _ := jv(s.DocumentVersionIDs)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO submissions (id, user_id, application_id, mode, adapter, status,
		 document_version_ids, approval_id, receipt_url, error_code, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.ID, s.UserID, s.ApplicationID, s.Mode, s.Adapter, s.Status,
		vids, s.ApprovalID, s.ReceiptURL, s.ErrorCode, s.CreatedAt, s.UpdatedAt)
	return err
}

func (r *SubmissionRepo) List(ctx context.Context, userID, applicationID uuid.UUID, page domain.PageRequest, limit int) ([]domain.Submission, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("application_id = %s", applicationID)
	c.cursor(page)
	sql, args := c.query(`id, user_id, application_id, mode, adapter, status, document_version_ids,
		approval_id, receipt_url, error_code, created_at, updated_at`, "submissions", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Submission{}
	for rows.Next() {
		s := domain.Submission{}
		var vids []byte
		if err := rows.Scan(&s.ID, &s.UserID, &s.ApplicationID, &s.Mode, &s.Adapter, &s.Status,
			&vids, &s.ApprovalID, &s.ReceiptURL, &s.ErrorCode, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if err := unj(vids, &s.DocumentVersionIDs); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
