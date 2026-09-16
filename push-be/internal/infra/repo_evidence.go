package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const evidenceCols = `id, user_id, revision, kind, title, source_text, source_url, skills,
	verification_status, provenance, supersedes_id, archived, created_at, updated_at`

type EvidenceRepo struct{ d *DB }

func NewEvidenceRepo(d *DB) *EvidenceRepo { return &EvidenceRepo{d: d} }

func scanEvidence(row interface{ Scan(...any) error }) (*domain.CareerEvidence, error) {
	e := &domain.CareerEvidence{}
	var skills, provenance []byte
	err := row.Scan(&e.ID, &e.UserID, &e.Revision, &e.Kind, &e.Title, &e.SourceText,
		&e.SourceURL, &skills, &e.VerificationStatus, &provenance, &e.SupersedesID,
		&e.Archived, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(skills, &e.Skills); err != nil {
		return nil, err
	}
	if err := unj(provenance, &e.Provenance); err != nil {
		return nil, err
	}
	if e.Skills == nil {
		e.Skills = []string{}
	}
	return e, nil
}

func (r *EvidenceRepo) Create(ctx context.Context, e *domain.CareerEvidence) error {
	skills, _ := jv(e.Skills)
	prov, _ := jv(e.Provenance)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO career_evidence (`+evidenceCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.ID, e.UserID, e.Revision, e.Kind, e.Title, e.SourceText, e.SourceURL,
		skills, e.VerificationStatus, prov, e.SupersedesID, e.Archived, e.CreatedAt, e.UpdatedAt)
	return err
}

func (r *EvidenceRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.CareerEvidence, error) {
	e, err := scanEvidence(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+evidenceCols+` FROM career_evidence WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return e, nil
}

func (r *EvidenceRepo) List(ctx context.Context, userID uuid.UUID, f domain.EvidenceFilter, limit int) ([]domain.CareerEvidence, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if f.Kind != nil {
		c.add("kind = %s", *f.Kind)
	}
	if f.Query != "" {
		c.add("(title ILIKE %s OR source_text ILIKE %s)", "%"+f.Query+"%", "%"+f.Query+"%")
	}
	c.cursor(f.Page)
	sql, args := c.query(evidenceCols, "career_evidence", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CareerEvidence{}
	for rows.Next() {
		e, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *EvidenceRepo) Update(ctx context.Context, e *domain.CareerEvidence, expectedRevision int64) error {
	skills, _ := jv(e.Skills)
	prov, _ := jv(e.Provenance)
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE career_evidence SET revision = revision + 1, kind = $3, title = $4, source_text = $5,
		 source_url = $6, skills = $7, verification_status = $8, provenance = $9,
		 supersedes_id = $10, archived = $11, updated_at = $12
		 WHERE id = $1 AND user_id = $2 AND revision = $13`,
		e.ID, e.UserID, e.Kind, e.Title, e.SourceText, e.SourceURL, skills,
		e.VerificationStatus, prov, e.SupersedesID, e.Archived, e.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "career_evidence", e.ID, e.UserID, expectedRevision)
	}
	return nil
}

func (r *EvidenceRepo) Count(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM career_evidence WHERE user_id = $1 AND NOT archived`, userID).Scan(&n)
	return n, err
}

type SourceRepo struct{ d *DB }

func NewSourceRepo(d *DB) *SourceRepo { return &SourceRepo{d: d} }

func (r *SourceRepo) Create(ctx context.Context, s *domain.Source) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO sources (id, user_id, file_name, mime_type, size, sha256, path, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		s.ID, s.UserID, s.FileName, s.MimeType, s.Size, s.SHA256, s.Path, s.Status, s.CreatedAt)
	return err
}

func (r *SourceRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Source, error) {
	s := &domain.Source{}
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT id, user_id, file_name, mime_type, size, sha256, path, status, created_at
		 FROM sources WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&s.ID, &s.UserID, &s.FileName, &s.MimeType, &s.Size, &s.SHA256, &s.Path, &s.Status, &s.CreatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	return s, nil
}

func (r *SourceRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.d.Q(ctx).Exec(ctx, `DELETE FROM sources WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SourceRepo) Referenced(ctx context.Context, userID, id uuid.UUID) (bool, error) {
	var n int
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM career_evidence WHERE user_id = $1 AND provenance->>'sourceId' = $2`,
		userID, id.String()).Scan(&n)
	return n > 0, err
}
