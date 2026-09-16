package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const interviewCols = `id, user_id, revision, application_id, title, scheduled_at, duration_minutes,
	event_id, evidence_ids, company_sources, notes, reflection, created_at, updated_at`

type InterviewRepo struct{ d *DB }

func NewInterviewRepo(d *DB) *InterviewRepo { return &InterviewRepo{d: d} }

func scanInterview(row interface{ Scan(...any) error }) (*domain.InterviewSession, error) {
	v := &domain.InterviewSession{}
	var eids, sources []byte
	err := row.Scan(&v.ID, &v.UserID, &v.Revision, &v.ApplicationID, &v.Title,
		&v.ScheduledAt, &v.DurationMinutes, &v.EventID, &eids, &sources,
		&v.Notes, &v.Reflection, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(eids, &v.EvidenceIDs); err != nil {
		return nil, err
	}
	if err := unj(sources, &v.CompanySources); err != nil {
		return nil, err
	}
	if v.EvidenceIDs == nil {
		v.EvidenceIDs = []uuid.UUID{}
	}
	if v.CompanySources == nil {
		v.CompanySources = []domain.CompanySource{}
	}
	return v, nil
}

func (r *InterviewRepo) Create(ctx context.Context, v *domain.InterviewSession) error {
	eids, _ := jv(v.EvidenceIDs)
	sources, _ := jv(v.CompanySources)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO interviews (`+interviewCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		v.ID, v.UserID, v.Revision, v.ApplicationID, v.Title, v.ScheduledAt,
		v.DurationMinutes, v.EventID, eids, sources, v.Notes, v.Reflection,
		v.CreatedAt, v.UpdatedAt)
	return err
}

func (r *InterviewRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.InterviewSession, error) {
	v, err := scanInterview(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+interviewCols+` FROM interviews WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return v, nil
}

func (r *InterviewRepo) List(ctx context.Context, userID uuid.UUID, f domain.InterviewFilter, limit int) ([]domain.InterviewSession, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if f.ApplicationID != nil {
		c.add("application_id = %s", *f.ApplicationID)
	}
	if f.From != nil {
		c.add("scheduled_at >= %s", *f.From)
	}
	if f.To != nil {
		c.add("scheduled_at <= %s", *f.To)
	}
	c.cursor(f.Page)
	sql, args := c.query(interviewCols, "interviews", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.InterviewSession{}
	for rows.Next() {
		v, err := scanInterview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *InterviewRepo) Update(ctx context.Context, v *domain.InterviewSession, expectedRevision int64) error {
	eids, _ := jv(v.EvidenceIDs)
	sources, _ := jv(v.CompanySources)
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE interviews SET revision = revision + 1, title = $3, scheduled_at = $4,
		 duration_minutes = $5, event_id = $6, evidence_ids = $7, company_sources = $8,
		 notes = $9, reflection = $10, updated_at = $11
		 WHERE id = $1 AND user_id = $2 AND revision = $12`,
		v.ID, v.UserID, v.Title, v.ScheduledAt, v.DurationMinutes, v.EventID,
		eids, sources, v.Notes, v.Reflection, v.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "interviews", v.ID, v.UserID, expectedRevision)
	}
	return nil
}

func (r *InterviewRepo) CreateCalendarEvent(ctx context.Context, e *domain.CalendarEvent) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO calendar_events (id, user_id, revision, application_id, type, title,
		 starts_at, ends_at, time_zone, source, external_id, notes, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.ID, e.UserID, e.Revision, e.ApplicationID, e.Type, e.Title,
		e.StartsAt, e.EndsAt, e.TimeZone, e.Source, e.ExternalID, e.Notes,
		e.CreatedAt, e.UpdatedAt)
	return err
}

func (r *InterviewRepo) UpdateCalendarEvent(ctx context.Context, e *domain.CalendarEvent) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE calendar_events SET title = coalesce(nullif($2,''), title), starts_at = $3,
		 ends_at = $4, updated_at = now() WHERE id = $1`,
		e.ID, e.Title, e.StartsAt, e.EndsAt)
	return err
}

func (r *InterviewRepo) CountByApplication(ctx context.Context, userID, applicationID uuid.UUID) (int, error) {
	var n int
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM interviews WHERE user_id = $1 AND application_id = $2`,
		userID, applicationID).Scan(&n)
	return n, err
}
