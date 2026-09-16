package presentation

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
)

func bindJSON(c echo.Context, dst any) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return domain.Validation("unreadable body")
	}
	if len(body) == 0 {
		return domain.Validation("request body required")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.Validation("invalid request body: " + err.Error())
	}
	if dec.More() {
		return domain.Validation("trailing data in body")
	}
	return nil
}

func paramID(c echo.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, domain.ValidationField(name, "must be a UUID")
	}
	return id, nil
}

func pageRequest(c echo.Context) (domain.PageRequest, error) {
	p := domain.PageRequest{}
	if l := c.QueryParam("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil || n < 1 {
			return p, domain.ValidationField("limit", "must be a positive integer")
		}
		if n > domain.MaxLimit {
			n = domain.MaxLimit
		}
		p.Limit = n
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		decoded, err := domain.DecodeCursor(cur)
		if err != nil {
			return p, domain.ValidationField("cursor", "invalid cursor")
		}
		p.Cursor = &decoded
	}
	return p, nil
}

func pageBody[T any](p domain.Page[T]) map[string]any {
	return map[string]any{
		"data": p.Items,
		"page": map[string]any{"nextCursor": p.NextCursor, "hasMore": p.HasMore},
	}
}

func data(c echo.Context, status int, v any) error {
	return c.JSON(status, map[string]any{"data": v})
}

func list(c echo.Context, p any) error {
	return c.JSON(http.StatusOK, p)
}

func userID(c echo.Context) uuid.UUID {
	if u, ok := c.Get("user").(*domain.User); ok {
		return u.ID
	}
	return uuid.Nil
}

func parseDate(s string) (*time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, domain.ValidationField("deadline", "must be YYYY-MM-DD")
	}
	return &t, nil
}

func optionalDate(raw json.RawMessage) (*time.Time, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	if string(raw) == "null" {
		return nil, true, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false, domain.ValidationField("deadline", "must be YYYY-MM-DD")
	}
	t, err := parseDate(s)
	return t, true, err
}
