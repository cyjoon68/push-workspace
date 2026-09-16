package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
)

type stubIdemStore struct {
	mu      sync.Mutex
	records map[string]*domain.IdempotencyRecord
}

func newStubIdemStore() *stubIdemStore {
	return &stubIdemStore{records: map[string]*domain.IdempotencyRecord{}}
}

func (s *stubIdemStore) key(userID uuid.UUID, method, path string, key uuid.UUID) string {
	return userID.String() + "|" + method + "|" + path + "|" + key.String()
}

func (s *stubIdemStore) InsertPending(ctx context.Context, rec *domain.IdempotencyRecord) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(rec.UserID, rec.Method, rec.Path, rec.Key)
	if _, exists := s.records[k]; exists {
		return false, nil
	}
	s.records[k] = rec
	return true, nil
}

func (s *stubIdemStore) Get(ctx context.Context, userID uuid.UUID, method, path string, key uuid.UUID) (*domain.IdempotencyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[s.key(userID, method, path, key)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return rec, nil
}

func (s *stubIdemStore) Complete(ctx context.Context, id uuid.UUID, status int, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.records {
		if rec.ID == id {
			rec.ResponseStatus = &status
			rec.ResponseBody = body
			return nil
		}
	}
	return domain.ErrNotFound
}

func (s *stubIdemStore) Delete(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, rec := range s.records {
		if rec.ID == id {
			delete(s.records, k)
			return nil
		}
	}
	return domain.ErrNotFound
}

func newIdemTestServer(store domain.IdempotencyStore, calls *int) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	mw := NewIdempotencyMiddleware(store).Handle
	e.POST("/api/v1/things", func(c echo.Context) error {
		*calls++
		return c.JSON(http.StatusCreated, map[string]any{"data": map[string]any{"n": *calls}})
	}, mw)
	return e
}

func doPost(e *echo.Echo, path, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("bad error body: %v", err)
	}
	return body.Error.Code
}

func TestIdempotencyKeyRequired(t *testing.T) {
	calls := 0
	e := newIdemTestServer(newStubIdemStore(), &calls)
	rec := doPost(e, "/api/v1/things", "", `{"a":1}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if errorCode(t, rec) != domain.CodeValidation {
		t.Fatal("want VALIDATION_ERROR")
	}
	if calls != 0 {
		t.Fatal("handler must not run")
	}
}

func TestIdempotencyKeyMustBeUUID(t *testing.T) {
	calls := 0
	e := newIdemTestServer(newStubIdemStore(), &calls)
	rec := doPost(e, "/api/v1/things", "not-a-uuid", `{"a":1}`)
	if rec.Code != http.StatusBadRequest || calls != 0 {
		t.Fatalf("status = %d calls = %d", rec.Code, calls)
	}
}

func TestIdempotencyReplayReturnsStored(t *testing.T) {
	calls := 0
	e := newIdemTestServer(newStubIdemStore(), &calls)
	key := uuid.NewString()
	first := doPost(e, "/api/v1/things", key, `{"a":1}`)
	if first.Code != http.StatusCreated || calls != 1 {
		t.Fatalf("first: status=%d calls=%d", first.Code, calls)
	}
	second := doPost(e, "/api/v1/things", key, `{"a":1}`)
	if second.Code != http.StatusCreated || calls != 1 {
		t.Fatalf("replay: status=%d calls=%d", second.Code, calls)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("bodies differ: %s vs %s", first.Body, second.Body)
	}
}

func TestIdempotencyConflictOnDifferentBody(t *testing.T) {
	calls := 0
	e := newIdemTestServer(newStubIdemStore(), &calls)
	key := uuid.NewString()
	doPost(e, "/api/v1/things", key, `{"a":1}`)
	rec := doPost(e, "/api/v1/things", key, `{"a":2}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
	if errorCode(t, rec) != domain.CodeIdempotencyConflict {
		t.Fatal("want IDEMPOTENCY_CONFLICT")
	}
	if calls != 1 {
		t.Fatal("handler ran twice")
	}
}

func TestIdempotencyDifferentKeyRunsAgain(t *testing.T) {
	calls := 0
	e := newIdemTestServer(newStubIdemStore(), &calls)
	doPost(e, "/api/v1/things", uuid.NewString(), `{"a":1}`)
	doPost(e, "/api/v1/things", uuid.NewString(), `{"a":1}`)
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestIdempotencyGetSkips(t *testing.T) {
	calls := 0
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	e.GET("/api/v1/things", func(c echo.Context) error {
		calls++
		return c.NoContent(http.StatusOK)
	}, NewIdempotencyMiddleware(newStubIdemStore()).Handle)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/things", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || calls != 1 {
		t.Fatalf("status=%d calls=%d", rec.Code, calls)
	}
}
