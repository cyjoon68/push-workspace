package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

const conversationCols = `id, user_id, revision, application_id, title, pinned, archived, created_at, updated_at`

const messageCols = `id, user_id, conversation_id, role, text, attachments, operation_id, created_at`

type ConversationRepo struct{ d *DB }

func NewConversationRepo(d *DB) *ConversationRepo { return &ConversationRepo{d: d} }

func scanConversation(row interface{ Scan(...any) error }) (*domain.Conversation, error) {
	c := &domain.Conversation{}
	err := row.Scan(&c.ID, &c.UserID, &c.Revision, &c.ApplicationID, &c.Title,
		&c.Pinned, &c.Archived, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func scanMessage(row interface{ Scan(...any) error }) (*domain.Message, error) {
	m := &domain.Message{}
	var attachments []byte
	err := row.Scan(&m.ID, &m.UserID, &m.ConversationID, &m.Role, &m.Text,
		&attachments, &m.OperationID, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := unj(attachments, &m.Attachments); err != nil {
		return nil, err
	}
	if m.Attachments == nil {
		m.Attachments = []domain.MessageAttachment{}
	}
	return m, nil
}

func (r *ConversationRepo) Create(ctx context.Context, c *domain.Conversation) error {
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO conversations (`+conversationCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		c.ID, c.UserID, c.Revision, c.ApplicationID, c.Title, c.Pinned, c.Archived,
		c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *ConversationRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Conversation, error) {
	c, err := scanConversation(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+conversationCols+` FROM conversations WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return c, nil
}

func (r *ConversationRepo) List(ctx context.Context, userID uuid.UUID, f domain.ConversationFilter, limit int) ([]domain.Conversation, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("archived = %s", false)
	if f.ApplicationID != nil {
		c.add("application_id = %s", *f.ApplicationID)
	}
	c.cursor(f.Page)
	sql, args := c.query(conversationCols, "conversations", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Conversation{}
	for rows.Next() {
		v, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *ConversationRepo) Update(ctx context.Context, c *domain.Conversation, expectedRevision int64) error {
	tag, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE conversations SET revision = revision + 1, title = $3, pinned = $4,
		 archived = $5, updated_at = $6
		 WHERE id = $1 AND user_id = $2 AND revision = $7`,
		c.ID, c.UserID, c.Title, c.Pinned, c.Archived, c.UpdatedAt, expectedRevision)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return r.d.revisionGuard(ctx, "conversations", c.ID, c.UserID, expectedRevision)
	}
	return nil
}

func (r *ConversationRepo) CreateMessage(ctx context.Context, m *domain.Message) error {
	attachments, _ := jv(m.Attachments)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO messages (`+messageCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		m.ID, m.UserID, m.ConversationID, m.Role, m.Text, attachments,
		m.OperationID, m.CreatedAt)
	return err
}

func (r *ConversationRepo) ListMessages(ctx context.Context, userID, conversationID uuid.UUID, page domain.PageRequest, limit int) ([]domain.Message, error) {
	c := &conds{}
	c.add("user_id = %s", userID)
	c.add("conversation_id = %s", conversationID)
	c.cursor(page)
	sql, args := c.query(messageCols, "messages", limit)
	rows, err := r.d.Q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}
