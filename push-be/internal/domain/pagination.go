package domain

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultLimit = 50
	MaxLimit     = 100
)

type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

func (c Cursor) Encode() string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return Cursor{}, fmt.Errorf("bad cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Cursor{}, err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return Cursor{}, err
	}
	return Cursor{CreatedAt: ts, ID: id}, nil
}

type PageRequest struct {
	Limit  int
	Cursor *Cursor
}

func (p PageRequest) EffectiveLimit() int {
	if p.Limit <= 0 {
		return DefaultLimit
	}
	if p.Limit > MaxLimit {
		return MaxLimit
	}
	return p.Limit
}

type Page[T any] struct {
	Items      []T
	NextCursor *string
	HasMore    bool
}

func NewPage[T any](items []T, limit int, cursorOf func(T) Cursor) Page[T] {
	p := Page[T]{Items: items}
	if len(items) > limit {
		p.Items = items[:limit]
		p.HasMore = true
		c := cursorOf(p.Items[len(p.Items)-1]).Encode()
		p.NextCursor = &c
	}
	if p.Items == nil {
		p.Items = []T{}
	}
	return p
}
