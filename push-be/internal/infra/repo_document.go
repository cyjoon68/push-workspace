package infra

import (
	"context"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const docCols = `id, user_id, revision, application_id, title, kind, template, language,
	status, latest_version_id, finalized_version_id, created_at, updated_at`

type DocumentRepo struct{ d *DB }

func NewDocumentRepo(d *DB) *DocumentRepo { return &DocumentRepo{d: d} }

func scanDocument(row interface{ Scan(...any) error }) (*domain.Document, error) {
	d := &domain.Document{}
	err := row.Scan(&d.ID, &d.UserID, &d.Revision, &d.ApplicationID, &d.Title, &d.Kind,
		&d.Template, &d.Language, &d.Status, &d.LatestVersionID, &d.FinalizedVersionID,
		&d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DocumentRepo) Create(ctx context.Context, d *domain.Document) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO documents (`+docCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		d.ID, d.UserID, d.Revision, d.ApplicationID, d.Title, d.Kind, d.Template, d.Language,
		d.Status, d.LatestVersionID, d.FinalizedVersionID, d.CreatedAt, d.UpdatedAt)
	return err
}

func (r *DocumentRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Document, error) {
	d, err := scanDocument(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+docCols+` FROM documents WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return d, nil
}

func (r *DocumentRepo) List(ctx context.Context, userID uuid.UUID, f domain.DocumentFilter, limit int) ([]domain.Document, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	if f.ApplicationID != nil {
		c.add("application_id = %s", *f.ApplicationID)
	}
	if f.Kind != nil {
		c.add("kind = %s", *f.Kind)
	}
	c.cursor(f.Page)
	sql, args := c.query(docCols, "documents", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *DocumentRepo) Update(ctx context.Context, d *domain.Document, expectedRevision int64) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE documents SET revision = revision + 1, title = $3, template = $4, language = $5,
		 status = $6, latest_version_id = $7, finalized_version_id = $8, updated_at = $9
		 WHERE id = $1 AND user_id = $2 AND revision = $10`,
		d.ID, d.UserID, d.Title, d.Template, d.Language, d.Status,
		d.LatestVersionID, d.FinalizedVersionID, d.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "documents", d.ID, d.UserID, expectedRevision)
	}
	return nil
}

const versionCols = `id, user_id, document_id, application_id, number, content, blocks, change_note, quality, created_at`

func scanVersion(row interface{ Scan(...any) error }) (*domain.DocumentVersion, error) {
	v := &domain.DocumentVersion{}
	var blocks, quality []byte
	err := row.Scan(&v.ID, &v.UserID, &v.DocumentID, &v.ApplicationID, &v.Number,
		&v.Content, &blocks, &v.ChangeNote, &quality, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(blocks, &v.Blocks); err != nil {
		return nil, err
	}
	if err := unj(quality, &v.Quality); err != nil {
		return nil, err
	}
	if v.Blocks == nil {
		v.Blocks = []domain.Block{}
	}
	return v, nil
}

func (r *DocumentRepo) CreateVersion(ctx context.Context, v *domain.DocumentVersion) error {
	blocks, _ := jv(v.Blocks)
	quality, _ := jv(v.Quality)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO document_versions (`+versionCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		v.ID, v.UserID, v.DocumentID, v.ApplicationID, v.Number, v.Content, blocks,
		v.ChangeNote, quality, v.CreatedAt)
	return err
}

func (r *DocumentRepo) GetVersion(ctx context.Context, userID, documentID, versionID uuid.UUID) (*domain.DocumentVersion, error) {
	q := `SELECT ` + versionCols + ` FROM document_versions WHERE id = $1 AND user_id = $2`
	args := []any{versionID, userID}
	if documentID != uuid.Nil {
		q += ` AND document_id = $3`
		args = append(args, documentID)
	}
	v, err := scanVersion(r.d.Q(ctx).QueryRow(ctx, q, args...))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return v, nil
}

func (r *DocumentRepo) GetVersionByID(ctx context.Context, userID, versionID uuid.UUID) (*domain.DocumentVersion, error) {
	return r.GetVersion(ctx, userID, uuid.Nil, versionID)
}

func (r *DocumentRepo) ListVersions(ctx context.Context, userID, documentID uuid.UUID, page domain.PageRequest, limit int) ([]domain.DocumentVersion, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("document_id = %s", documentID)
	c.cursor(page)
	sql, args := c.query(versionCols, "document_versions", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.DocumentVersion{}
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *DocumentRepo) NextVersionNumber(ctx context.Context, documentID uuid.UUID) (int, error) {
	var n *int
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT max(number) FROM document_versions WHERE document_id = $1`, documentID).Scan(&n)
	if err != nil {
		return 0, err
	}
	if n == nil {
		return 1, nil
	}
	return *n + 1, nil
}

func (r *DocumentRepo) CreateProposal(ctx context.Context, p *domain.RevisionProposal) error {
	sel, _ := jv(p.Selection)
	refs, _ := jv(p.EvidenceRefs)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO revision_proposals (id, user_id, document_id, source_version_id, source_revision,
		 selection, replacement, evidence_refs, claim_status, applied_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		p.ID, p.UserID, p.DocumentID, p.SourceVersionID, p.SourceRevision,
		sel, p.Replacement, refs, p.ClaimStatus, p.AppliedAt, p.CreatedAt)
	return err
}

func (r *DocumentRepo) GetProposal(ctx context.Context, userID, documentID, proposalID uuid.UUID) (*domain.RevisionProposal, error) {
	p := &domain.RevisionProposal{}
	var sel, refs []byte
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT id, user_id, document_id, source_version_id, source_revision, selection,
		 replacement, evidence_refs, claim_status, applied_at, created_at
		 FROM revision_proposals WHERE id = $1 AND document_id = $2 AND user_id = $3`,
		proposalID, documentID, userID).
		Scan(&p.ID, &p.UserID, &p.DocumentID, &p.SourceVersionID, &p.SourceRevision,
			&sel, &p.Replacement, &refs, &p.ClaimStatus, &p.AppliedAt, &p.CreatedAt)
	if err != nil {
		return nil, mapGetErr(err)
	}
	if err := unj(sel, &p.Selection); err != nil {
		return nil, err
	}
	if err := unj(refs, &p.EvidenceRefs); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *DocumentRepo) MarkProposalApplied(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE revision_proposals SET applied_at = $2 WHERE id = $1`, id, at)
	return err
}

const exportCols = `id, user_id, document_id, version_id, format, template, language, content,
	blocks, content_hash, status, renderer_version, result, created_at, updated_at`

func scanExport(row interface{ Scan(...any) error }) (*domain.DocumentExport, error) {
	e := &domain.DocumentExport{}
	var blocks, result []byte
	err := row.Scan(&e.ID, &e.UserID, &e.DocumentID, &e.VersionID, &e.Format, &e.Template,
		&e.Language, &e.Content, &blocks, &e.ContentHash, &e.Status, &e.RendererVersion,
		&result, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(blocks, &e.Blocks); err != nil {
		return nil, err
	}
	if len(result) > 0 {
		res := &domain.ExportResult{}
		if err := unj(result, res); err != nil {
			return nil, err
		}
		e.Result = res
	}
	return e, nil
}

func (r *DocumentRepo) CreateExport(ctx context.Context, e *domain.DocumentExport) error {
	blocks, _ := jv(e.Blocks)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO document_exports (`+exportCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		e.ID, e.UserID, e.DocumentID, e.VersionID, e.Format, e.Template, e.Language,
		e.Content, blocks, e.ContentHash, e.Status, e.RendererVersion, nil, e.CreatedAt, e.UpdatedAt)
	return err
}

func (r *DocumentRepo) GetExport(ctx context.Context, userID, documentID, exportID uuid.UUID) (*domain.DocumentExport, error) {
	e, err := scanExport(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+exportCols+` FROM document_exports WHERE id = $1 AND document_id = $2 AND user_id = $3`,
		exportID, documentID, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return e, nil
}

func (r *DocumentRepo) ListExports(ctx context.Context, userID, documentID uuid.UUID, page domain.PageRequest, limit int) ([]domain.DocumentExport, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("document_id = %s", documentID)
	c.cursor(page)
	sql, args := c.query(exportCols, "document_exports", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.DocumentExport{}
	for rows.Next() {
		e, err := scanExport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *DocumentRepo) UpdateExportResult(ctx context.Context, e *domain.DocumentExport) error {
	res, _ := jv(e.Result)
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE document_exports SET status = $2, result = $3, updated_at = $4 WHERE id = $1`,
		e.ID, e.Status, res, e.UpdatedAt)
	return err
}

func (r *DocumentRepo) CountFinalizedByApplication(ctx context.Context, userID, applicationID uuid.UUID) (int, error) {
	var n int
	err := r.d.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM documents WHERE user_id = $1 AND application_id = $2 AND status = 'FINALIZED'`,
		userID, applicationID).Scan(&n)
	return n, err
}
