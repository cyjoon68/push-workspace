package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const calendarEventCols = `id, user_id, revision, application_id, type, title,
	starts_at, ends_at, time_zone, source, external_id, notes, created_at, updated_at`

type CalendarRepo struct{ d *DB }

func NewCalendarRepo(d *DB) *CalendarRepo { return &CalendarRepo{d: d} }

func scanCalendarEvent(row interface{ Scan(...any) error }) (*domain.CalendarEvent, error) {
	e := &domain.CalendarEvent{}
	err := row.Scan(&e.ID, &e.UserID, &e.Revision, &e.ApplicationID, &e.Type, &e.Title,
		&e.StartsAt, &e.EndsAt, &e.TimeZone, &e.Source, &e.ExternalID, &e.Notes,
		&e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *CalendarRepo) CreateCalendarEvent(ctx context.Context, e *domain.CalendarEvent) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO calendar_events (`+calendarEventCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.ID, e.UserID, e.Revision, e.ApplicationID, e.Type, e.Title,
		e.StartsAt, e.EndsAt, e.TimeZone, e.Source, e.ExternalID, e.Notes,
		e.CreatedAt, e.UpdatedAt)
	return err
}

func (r *CalendarRepo) GetCalendarEvent(ctx context.Context, userID, id uuid.UUID) (*domain.CalendarEvent, error) {
	e, err := scanCalendarEvent(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+calendarEventCols+` FROM calendar_events WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return e, nil
}

func (r *CalendarRepo) ListCalendarEvents(ctx context.Context, userID uuid.UUID, f domain.CalendarEventFilter, limit int) ([]domain.CalendarEvent, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if f.ApplicationID != nil {
		c.add("application_id = %s", *f.ApplicationID)
	}
	if f.From != nil {
		c.add("starts_at >= %s", *f.From)
	}
	if f.To != nil {
		c.add("starts_at <= %s", *f.To)
	}
	if f.Source != nil {
		c.add("source = %s", *f.Source)
	}
	c.cursor(f.Page)
	sql, args := c.query(calendarEventCols, "calendar_events", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CalendarEvent{}
	for rows.Next() {
		e, err := scanCalendarEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *CalendarRepo) PatchCalendarEvent(ctx context.Context, e *domain.CalendarEvent, expectedRevision int64) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE calendar_events SET revision = revision + 1, title = $3, starts_at = $4,
		 ends_at = $5, time_zone = $6, notes = $7, updated_at = $8
		 WHERE id = $1 AND user_id = $2 AND revision = $9`,
		e.ID, e.UserID, e.Title, e.StartsAt, e.EndsAt, e.TimeZone, e.Notes, e.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "calendar_events", e.ID, e.UserID, expectedRevision)
	}
	return nil
}

func (r *CalendarRepo) DeleteCalendarEvent(ctx context.Context, userID, id uuid.UUID, expectedRevision int64) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`DELETE FROM calendar_events WHERE id = $1 AND user_id = $2 AND revision = $3`,
		id, userID, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "calendar_events", id, userID, expectedRevision)
	}
	return nil
}

func (r *CalendarRepo) UpsertExternalCalendarEvent(ctx context.Context, e *domain.CalendarEvent) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO calendar_events (`+calendarEventCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 ON CONFLICT (user_id, external_id) WHERE external_id IS NOT NULL DO UPDATE
		 SET title=$6, starts_at=$7, ends_at=$8, time_zone=$9, notes=$12,
		 revision=calendar_events.revision+1, updated_at=$14`,
		e.ID, e.UserID, e.Revision, e.ApplicationID, e.Type, e.Title,
		e.StartsAt, e.EndsAt, e.TimeZone, e.Source, e.ExternalID, e.Notes,
		e.CreatedAt, e.UpdatedAt)
	return err
}

func (r *CalendarRepo) DeleteExternalCalendarEvent(ctx context.Context, userID uuid.UUID, externalID string) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`DELETE FROM calendar_events WHERE user_id = $1 AND external_id = $2 AND source = 'GOOGLE'`,
		userID, externalID)
	return err
}
