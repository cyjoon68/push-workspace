package presentation

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func requestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get("X-Request-Id")
			if id == "" || len(id) > 128 {
				id = "req_" + uuid.NewString()
			}
			c.Set("requestId", id)
			c.Response().Header().Set("X-Request-Id", id)
			return next(c)
		}
	}
}

func bodyLimitMiddleware(limit int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			if r.ContentLength > limit {
				return domain.PayloadTooLarge()
			}
			r.Body = http.MaxBytesReader(c.Response(), r.Body, limit)
			return next(c)
		}
	}
}

func authMiddleware(auth *usecase.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				return domain.Unauthenticated("missing bearer token")
			}
			user, err := auth.ResolveAccessToken(c.Request().Context(), strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				return err
			}
			c.Set("user", user)
			return next(c)
		}
	}
}

type bodyCaptureWriter struct {
	http.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *bodyCaptureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	if w.body != nil {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

var idempotencyExempt = map[string]bool{
	"/api/v1/auth/exchange":   true,
	"/api/v1/auth/refresh":    true,
	"/api/v1/auth/logout":     true,
	"/api/v1/billing/webhook": true,
}

type IdempotencyMiddleware struct {
	store domain.IdempotencyStore
}

func NewIdempotencyMiddleware(store domain.IdempotencyStore) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{store: store}
}

func (m *IdempotencyMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		r := c.Request()
		if r.Method != http.MethodPost || idempotencyExempt[r.URL.Path] {
			return next(c)
		}
		keyHeader := r.Header.Get("Idempotency-Key")
		key, err := uuid.Parse(keyHeader)
		if err != nil {
			return domain.ValidationField("Idempotency-Key", "must be a UUID")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return domain.PayloadTooLarge()
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		hash := domain.HashBytes(body)
		uid := userID(c)
		path := r.URL.Path
		rec := &domain.IdempotencyRecord{
			ID: uuid.New(), UserID: uid, Key: key, Method: r.Method,
			Path: path, RequestHash: hash, CreatedAt: time.Now().UTC(),
		}
		inserted, err := m.store.InsertPending(r.Context(), rec)
		if err != nil {
			return domain.Internal()
		}
		if !inserted {
			existing, err := m.store.Get(r.Context(), uid, r.Method, path, key)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					return domain.Internal()
				}
				return domain.Internal()
			}
			if existing.RequestHash != hash {
				return domain.IdempotencyConflict()
			}
			if existing.ResponseStatus == nil {
				return domain.IdempotencyConflict()
			}
			c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
			return c.Blob(*existing.ResponseStatus, "application/json; charset=utf-8", existing.ResponseBody)
		}
		buf := &bytes.Buffer{}
		orig := c.Response().Writer
		cw := &bodyCaptureWriter{ResponseWriter: orig, body: buf, status: http.StatusOK}
		c.Response().Writer = cw
		handlerErr := next(c)
		c.Response().Writer = orig
		if handlerErr != nil {
			_ = m.store.Delete(r.Context(), rec.ID)
			return handlerErr
		}
		status := cw.status
		if status >= 200 && status < 300 {
			_ = m.store.Complete(r.Context(), rec.ID, status, buf.Bytes())
		} else {
			_ = m.store.Delete(r.Context(), rec.ID)
		}
		return nil
	}
}
