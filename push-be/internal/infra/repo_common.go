package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"push-be/internal/domain"
)

type conds struct {
	where []string
	args  []any
}

func (c *conds) add(clause string, vals ...any) {
	c.args = append(c.args, vals...)
	c.where = append(c.where, fmt.Sprintf(clause, placeholders(len(c.args)-len(vals)+1, len(vals))...))
}

func placeholders(start, n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = fmt.Sprintf("$%d", start+i)
	}
	return out
}

func (c *conds) cursor(page domain.PageRequest) {
	if page.Cursor != nil {
		c.args = append(c.args, page.Cursor.CreatedAt, page.Cursor.ID)
		c.where = append(c.where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", len(c.args)-1, len(c.args)))
	}
}

func (c *conds) query(cols, table string, limit int) (string, []any) {
	c.args = append(c.args, limit)
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d",
		cols, table, strings.Join(c.where, " AND "), len(c.args)), c.args
}

func jv(v any) ([]byte, error) {
	if v == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return []byte("[]"), nil
	}
	return b, nil
}

func unj[T any](b []byte, out *T) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}

func mapGetErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func (d *DB) revisionGuard(ctx context.Context, table string, id, userID uuid.UUID, expected int64) error {
	var cur int64
	err := d.Q(ctx).QueryRow(ctx,
		fmt.Sprintf("SELECT revision FROM %s WHERE id = $1 AND user_id = $2", table), id, userID).Scan(&cur)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	return &domain.Conflict{Current: cur}
}

type execResult interface {
	RowsAffected() int64
}
