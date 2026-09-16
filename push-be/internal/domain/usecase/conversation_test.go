package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type stubConversationStore struct {
	conv      *domain.Conversation
	messages  []domain.Message
	created   []*domain.Conversation
	updateErr error
}

func (s *stubConversationStore) Create(ctx context.Context, c *domain.Conversation) error {
	s.created = append(s.created, c)
	return nil
}

func (s *stubConversationStore) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Conversation, error) {
	if s.conv == nil || s.conv.ID != id || s.conv.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return s.conv, nil
}

func (s *stubConversationStore) List(ctx context.Context, userID uuid.UUID, f domain.ConversationFilter, limit int) ([]domain.Conversation, error) {
	return nil, nil
}

func (s *stubConversationStore) Update(ctx context.Context, c *domain.Conversation, expectedRevision int64) error {
	return s.updateErr
}

func (s *stubConversationStore) CreateMessage(ctx context.Context, m *domain.Message) error {
	s.messages = append(s.messages, *m)
	return nil
}

func (s *stubConversationStore) ListMessages(ctx context.Context, userID, conversationID uuid.UUID, page domain.PageRequest, limit int) ([]domain.Message, error) {
	return s.messages, nil
}

type stubDocumentStore struct {
	doc     *domain.Document
	version *domain.DocumentVersion
}

func (s *stubDocumentStore) Create(ctx context.Context, d *domain.Document) error { return nil }

func (s *stubDocumentStore) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Document, error) {
	if s.doc == nil || s.doc.ID != id || s.doc.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return s.doc, nil
}

func (s *stubDocumentStore) List(ctx context.Context, userID uuid.UUID, f domain.DocumentFilter, limit int) ([]domain.Document, error) {
	return nil, nil
}

func (s *stubDocumentStore) Update(ctx context.Context, d *domain.Document, expectedRevision int64) error {
	return nil
}

func (s *stubDocumentStore) CreateVersion(ctx context.Context, v *domain.DocumentVersion) error {
	return nil
}

func (s *stubDocumentStore) GetVersion(ctx context.Context, userID, documentID, versionID uuid.UUID) (*domain.DocumentVersion, error) {
	if s.version == nil || s.version.ID != versionID || s.version.DocumentID != documentID {
		return nil, domain.ErrNotFound
	}
	return s.version, nil
}

func (s *stubDocumentStore) GetVersionByID(ctx context.Context, userID, versionID uuid.UUID) (*domain.DocumentVersion, error) {
	if s.version == nil || s.version.ID != versionID || s.version.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return s.version, nil
}

func (s *stubDocumentStore) ListVersions(ctx context.Context, userID, documentID uuid.UUID, page domain.PageRequest, limit int) ([]domain.DocumentVersion, error) {
	return nil, nil
}

func (s *stubDocumentStore) NextVersionNumber(ctx context.Context, documentID uuid.UUID) (int, error) {
	return 0, nil
}

func (s *stubDocumentStore) CreateProposal(ctx context.Context, p *domain.RevisionProposal) error {
	return nil
}

func (s *stubDocumentStore) GetProposal(ctx context.Context, userID, documentID, proposalID uuid.UUID) (*domain.RevisionProposal, error) {
	return nil, domain.ErrNotFound
}

func (s *stubDocumentStore) MarkProposalApplied(ctx context.Context, id uuid.UUID, at time.Time) error {
	return nil
}

func (s *stubDocumentStore) CreateExport(ctx context.Context, e *domain.DocumentExport) error {
	return nil
}

func (s *stubDocumentStore) GetExport(ctx context.Context, userID, documentID, exportID uuid.UUID) (*domain.DocumentExport, error) {
	return nil, domain.ErrNotFound
}

func (s *stubDocumentStore) ListExports(ctx context.Context, userID, documentID uuid.UUID, page domain.PageRequest, limit int) ([]domain.DocumentExport, error) {
	return nil, nil
}

func (s *stubDocumentStore) UpdateExportResult(ctx context.Context, e *domain.DocumentExport) error {
	return nil
}

func (s *stubDocumentStore) CountFinalizedByApplication(ctx context.Context, userID, applicationID uuid.UUID) (int, error) {
	return 0, nil
}

type stubEvidenceStore struct {
	items map[uuid.UUID]*domain.CareerEvidence
}

func (s *stubEvidenceStore) Create(ctx context.Context, e *domain.CareerEvidence) error {
	return nil
}

func (s *stubEvidenceStore) Get(ctx context.Context, userID, id uuid.UUID) (*domain.CareerEvidence, error) {
	if e, ok := s.items[id]; ok && e.UserID == userID {
		return e, nil
	}
	return nil, domain.ErrNotFound
}

func (s *stubEvidenceStore) List(ctx context.Context, userID uuid.UUID, f domain.EvidenceFilter, limit int) ([]domain.CareerEvidence, error) {
	return nil, nil
}

func (s *stubEvidenceStore) Update(ctx context.Context, e *domain.CareerEvidence, expectedRevision int64) error {
	return nil
}

func (s *stubEvidenceStore) Count(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}

type stubOperationStore struct {
	created []*domain.Operation
}

func (s *stubOperationStore) Create(ctx context.Context, o *domain.Operation) error {
	s.created = append(s.created, o)
	return nil
}

func (s *stubOperationStore) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Operation, error) {
	return nil, domain.ErrNotFound
}

func (s *stubOperationStore) Update(ctx context.Context, o *domain.Operation) error {
	return nil
}

func newConversationService(convs *stubConversationStore, apps domain.ApplicationStore, docs *stubDocumentStore, ev *stubEvidenceStore, ops *stubOperationStore) *ConversationService {
	return NewConversationService(convs, apps, docs, ev, ops, noopUoW{}, &AIGate{}, &stubAiUsageStore{})
}

func TestConversationCreateRequiresApplication(t *testing.T) {
	svc := newConversationService(&stubConversationStore{}, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{}, &stubOperationStore{})
	appID := uuid.New()
	_, err := svc.Create(context.Background(), uuid.New(), CreateConversationInput{ApplicationID: &appID})
	if errCode(t, err) != domain.CodeNotFound {
		t.Fatalf("want NOT_FOUND, got %v", err)
	}
}

func TestConversationCreateWithoutApplication(t *testing.T) {
	store := &stubConversationStore{}
	svc := newConversationService(store, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{}, &stubOperationStore{})
	c, err := svc.Create(context.Background(), uuid.New(), CreateConversationInput{Title: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Revision != 1 || c.Title != "hello" || c.ApplicationID != nil {
		t.Fatalf("got %+v", c)
	}
	if len(store.created) != 1 {
		t.Fatal("conversation not persisted")
	}
}

func TestConversationPatchConflict(t *testing.T) {
	userID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 3}
	svc := newConversationService(&stubConversationStore{conv: conv}, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{}, &stubOperationStore{})
	pinned := true
	_, err := svc.Patch(context.Background(), userID, conv.ID, PatchConversationInput{
		ExpectedRevision: 1, Pinned: &pinned,
	})
	if errCode(t, err) != domain.CodeRevisionConflict {
		t.Fatalf("want REVISION_CONFLICT, got %v", err)
	}
}

func TestPostMessageRejectsBadAccessMode(t *testing.T) {
	userID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1}
	svc := newConversationService(&stubConversationStore{conv: conv}, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{}, &stubOperationStore{})
	_, err := svc.PostMessage(context.Background(), userID, conv.ID, PostMessageInput{
		Text: "hi", AccessMode: "YOLO", Context: MessageContext{},
	})
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}

func TestPostMessageArchivedConversation(t *testing.T) {
	userID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1, Archived: true}
	svc := newConversationService(&stubConversationStore{conv: conv}, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{}, &stubOperationStore{})
	_, err := svc.PostMessage(context.Background(), userID, conv.ID, PostMessageInput{
		Text: "hi", AccessMode: domain.AccessSuggest, Context: MessageContext{},
	})
	if errCode(t, err) != domain.CodeInvalidTransition {
		t.Fatalf("want INVALID_TRANSITION, got %v", err)
	}
}

func TestPostMessageEvidenceOwnership(t *testing.T) {
	userID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1}
	svc := newConversationService(&stubConversationStore{conv: conv}, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{items: map[uuid.UUID]*domain.CareerEvidence{}}, &stubOperationStore{})
	_, err := svc.PostMessage(context.Background(), userID, conv.ID, PostMessageInput{
		Text: "hi", AccessMode: domain.AccessSuggest,
		Context: MessageContext{EvidenceIDs: []uuid.UUID{uuid.New()}},
	})
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}

func TestPostMessageDocumentWithoutVersion(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1, ApplicationID: &appID}
	doc := &domain.Document{ID: uuid.New(), UserID: userID, ApplicationID: appID}
	svc := newConversationService(&stubConversationStore{conv: conv}, &stubApplicationStore{}, &stubDocumentStore{doc: doc}, &stubEvidenceStore{}, &stubOperationStore{})
	_, err := svc.PostMessage(context.Background(), userID, conv.ID, PostMessageInput{
		Text: "hi", AccessMode: domain.AccessSuggest,
		Context: MessageContext{DocumentID: &doc.ID},
	})
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}

func TestPostMessageDocumentScopeMismatch(t *testing.T) {
	userID := uuid.New()
	appID, otherAppID := uuid.New(), uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1, ApplicationID: &appID}
	versionID := uuid.New()
	doc := &domain.Document{ID: uuid.New(), UserID: userID, ApplicationID: otherAppID, LatestVersionID: &versionID}
	svc := newConversationService(&stubConversationStore{conv: conv}, &stubApplicationStore{}, &stubDocumentStore{doc: doc}, &stubEvidenceStore{}, &stubOperationStore{})
	_, err := svc.PostMessage(context.Background(), userID, conv.ID, PostMessageInput{
		Text: "hi", AccessMode: domain.AccessSuggest,
		Context: MessageContext{DocumentID: &doc.ID},
	})
	if errCode(t, err) != domain.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
}

func TestPostMessageSuccessCreatesOperation(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1, ApplicationID: &appID}
	versionID := uuid.New()
	doc := &domain.Document{ID: uuid.New(), UserID: userID, ApplicationID: appID, Title: "resume", LatestVersionID: &versionID}
	version := &domain.DocumentVersion{ID: versionID, UserID: userID, DocumentID: doc.ID, ApplicationID: appID}
	evID := uuid.New()
	evStore := &stubEvidenceStore{items: map[uuid.UUID]*domain.CareerEvidence{
		evID: {ID: evID, UserID: userID, Title: "ev"},
	}}
	convs := &stubConversationStore{conv: conv}
	ops := &stubOperationStore{}
	svc := newConversationService(convs, &stubApplicationStore{}, &stubDocumentStore{doc: doc, version: version}, evStore, ops)
	op, err := svc.PostMessage(context.Background(), userID, conv.ID, PostMessageInput{
		Text: "도와줘", AccessMode: domain.AccessConfirm,
		Context: MessageContext{DocumentID: &doc.ID, EvidenceIDs: []uuid.UUID{evID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if op.Status != domain.OpSucceeded || op.Type != domain.OpChatMessage {
		t.Fatalf("got status=%s type=%s", op.Status, op.Type)
	}
	if len(convs.messages) != 2 {
		t.Fatalf("want 2 messages, got %d", len(convs.messages))
	}
	userMsg, aiMsg := convs.messages[0], convs.messages[1]
	if userMsg.Role != domain.RoleUser || aiMsg.Role != domain.RoleAssistant {
		t.Fatalf("roles: %s %s", userMsg.Role, aiMsg.Role)
	}
	if userMsg.OperationID == nil || *userMsg.OperationID != op.ID {
		t.Fatal("user message missing operationId")
	}
	if len(userMsg.Attachments) != 2 {
		t.Fatalf("want 2 attachments, got %d", len(userMsg.Attachments))
	}
	if userMsg.Attachments[0].Type != domain.AttachDocumentVersion || userMsg.Attachments[0].ID != versionID {
		t.Fatalf("attachment: %+v", userMsg.Attachments[0])
	}
	if len(ops.created) != 1 {
		t.Fatal("operation not persisted")
	}
}

func TestConversationArchive(t *testing.T) {
	userID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1}
	store := &stubConversationStore{conv: conv}
	svc := newConversationService(store, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{}, &stubOperationStore{})
	out, err := svc.Archive(context.Background(), userID, conv.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Archived || out.Revision != 2 {
		t.Fatalf("got archived=%v revision=%d", out.Archived, out.Revision)
	}
}
