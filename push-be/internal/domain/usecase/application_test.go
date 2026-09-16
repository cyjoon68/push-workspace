package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type noopUoW struct{}

func (noopUoW) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type stubApplicationStore struct {
	app         *domain.Application
	updateErr   error
	events      []domain.ApplicationEvent
	updateCalls int
}

func (s *stubApplicationStore) Create(ctx context.Context, a *domain.Application) error {
	return nil
}

func (s *stubApplicationStore) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Application, error) {
	if s.app == nil || s.app.ID != id || s.app.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return s.app, nil
}

func (s *stubApplicationStore) List(ctx context.Context, userID uuid.UUID, f domain.ApplicationFilter, limit int) ([]domain.Application, error) {
	return nil, nil
}

func (s *stubApplicationStore) Update(ctx context.Context, a *domain.Application, expectedRevision int64) error {
	s.updateCalls++
	if s.updateErr != nil {
		return s.updateErr
	}
	return nil
}

func (s *stubApplicationStore) AddEvent(ctx context.Context, e *domain.ApplicationEvent) error {
	s.events = append(s.events, *e)
	return nil
}

func (s *stubApplicationStore) ListEvents(ctx context.Context, userID, applicationID uuid.UUID, page domain.PageRequest, limit int) ([]domain.ApplicationEvent, error) {
	return s.events, nil
}

func newAppService(store *stubApplicationStore) *ApplicationService {
	return NewApplicationService(store, nil, nil, nil, nil, nil, nil, nil, noopUoW{}, map[string]bool{})
}

func errCode(t *testing.T, err error) string {
	t.Helper()
	de, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("expected *domain.Error, got %T: %v", err, err)
	}
	return de.Code
}

func TestPatchInvalidTransition(t *testing.T) {
	userID := uuid.New()
	app := &domain.Application{
		ID: uuid.New(), UserID: userID, Revision: 1,
		Stage: domain.StageDiscovered,
	}
	svc := newAppService(&stubApplicationStore{app: app})
	to := domain.StageScreening
	_, err := svc.Patch(context.Background(), userID, app.ID, PatchApplicationInput{
		ExpectedRevision: 1, Stage: &to,
	})
	if errCode(t, err) != domain.CodeInvalidTransition {
		t.Fatalf("want INVALID_TRANSITION, got %v", err)
	}
	to = domain.StageApplied
	_, err = svc.Patch(context.Background(), userID, app.ID, PatchApplicationInput{
		ExpectedRevision: 1, Stage: &to,
	})
	if errCode(t, err) != domain.CodeInvalidTransition {
		t.Fatalf("want INVALID_TRANSITION for direct APPLIED, got %v", err)
	}
}

func TestPatchValidTransitionRecordsEvent(t *testing.T) {
	userID := uuid.New()
	app := &domain.Application{
		ID: uuid.New(), UserID: userID, Revision: 1,
		Stage: domain.StageDiscovered,
	}
	store := &stubApplicationStore{app: app}
	svc := newAppService(store)
	to := domain.StagePreparing
	out, err := svc.Patch(context.Background(), userID, app.ID, PatchApplicationInput{
		ExpectedRevision: 1, Stage: &to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Stage != domain.StagePreparing || out.Revision != 2 {
		t.Fatalf("got stage=%s revision=%d", out.Stage, out.Revision)
	}
	if len(store.events) != 1 || store.events[0].Type != domain.EventStageChanged {
		t.Fatal("expected STAGE_CHANGED event")
	}
}

func TestPatchRevisionConflict(t *testing.T) {
	userID := uuid.New()
	app := &domain.Application{
		ID: uuid.New(), UserID: userID, Revision: 5,
		Stage: domain.StageDiscovered,
	}
	svc := newAppService(&stubApplicationStore{app: app})
	_, err := svc.Patch(context.Background(), userID, app.ID, PatchApplicationInput{
		ExpectedRevision: 3,
	})
	if errCode(t, err) != domain.CodeRevisionConflict {
		t.Fatalf("want REVISION_CONFLICT, got %v", err)
	}
	de := err.(*domain.Error)
	if de.Details["currentRevision"] != int64(5) {
		t.Fatalf("currentRevision = %v", de.Details["currentRevision"])
	}
}

func TestPatchRevisionConflictFromStore(t *testing.T) {
	userID := uuid.New()
	app := &domain.Application{
		ID: uuid.New(), UserID: userID, Revision: 2,
		Stage: domain.StageDiscovered,
	}
	store := &stubApplicationStore{app: app, updateErr: &domain.Conflict{Current: 7}}
	svc := newAppService(store)
	_, err := svc.Patch(context.Background(), userID, app.ID, PatchApplicationInput{
		ExpectedRevision: 2,
	})
	if errCode(t, err) != domain.CodeRevisionConflict {
		t.Fatalf("want REVISION_CONFLICT, got %v", err)
	}
	if err.(*domain.Error).Details["currentRevision"] != int64(7) {
		t.Fatal("currentRevision mismatch")
	}
}

func TestPatchOwnershipIsolation(t *testing.T) {
	app := &domain.Application{
		ID: uuid.New(), UserID: uuid.New(), Revision: 1,
		Stage: domain.StageDiscovered,
	}
	svc := newAppService(&stubApplicationStore{app: app})
	_, err := svc.Patch(context.Background(), uuid.New(), app.ID, PatchApplicationInput{
		ExpectedRevision: 1,
	})
	if errCode(t, err) != domain.CodeNotFound {
		t.Fatalf("want NOT_FOUND, got %v", err)
	}
}

func TestImportRequiresConfirmation(t *testing.T) {
	svc := newAppService(&stubApplicationStore{})
	_, err := svc.Import(context.Background(), uuid.New(), ImportApplicationInput{
		JobID: uuid.New(), Stage: domain.StageApplied, AppliedAt: time.Now(),
	})
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}

func TestImportRejectsDiscovered(t *testing.T) {
	svc := newAppService(&stubApplicationStore{})
	_, err := svc.Import(context.Background(), uuid.New(), ImportApplicationInput{
		JobID: uuid.New(), Stage: domain.StageDiscovered, AppliedAt: time.Now(), Confirmed: true,
	})
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}
