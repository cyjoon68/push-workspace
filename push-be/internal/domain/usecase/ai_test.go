package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type stubChatCompleter struct {
	text     string
	in       int
	out      int
	err      error
	gotKey   string
	gotModel string
	gotUser  string
	gotSys   string
}

func (s *stubChatCompleter) Chat(ctx context.Context, apiKey, model, system, user string) (*domain.AICompletion, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.gotKey, s.gotModel, s.gotSys, s.gotUser = apiKey, model, system, user
	return &domain.AICompletion{Text: s.text, InputTokens: s.in, OutputTokens: s.out}, nil
}

type stubCipher struct {
	plaintext string
	err       error
	gotAAD    string
}

func (s *stubCipher) Decrypt(ciphertext, nonce []byte, userID uuid.UUID, provider string) (string, error) {
	s.gotAAD = userID.String() + "|" + provider
	return s.plaintext, s.err
}

type stubAiKeyStore struct {
	key *domain.AiKey
}

func (s *stubAiKeyStore) Put(ctx context.Context, k *domain.AiKey) error { return nil }

func (s *stubAiKeyStore) Get(ctx context.Context, userID uuid.UUID, provider string) (*domain.AiKey, error) {
	if s.key == nil || s.key.Provider != provider || s.key.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return s.key, nil
}

func (s *stubAiKeyStore) List(ctx context.Context, userID uuid.UUID) ([]domain.AiKey, error) {
	return nil, nil
}

func (s *stubAiKeyStore) Delete(ctx context.Context, userID uuid.UUID, provider string) error {
	return nil
}

type stubAiUsageStore struct {
	items []domain.AiUsage
}

func (s *stubAiUsageStore) Create(ctx context.Context, u *domain.AiUsage) error {
	s.items = append(s.items, *u)
	return nil
}

func (s *stubAiUsageStore) List(ctx context.Context, userID uuid.UUID, f domain.AiUsageFilter, limit int) ([]domain.AiUsage, error) {
	out := []domain.AiUsage{}
	for _, u := range s.items {
		if u.UserID == userID {
			out = append(out, u)
		}
	}
	return out, nil
}

func managedAI() *domain.AiOptions {
	return &domain.AiOptions{Provider: "OPENAI", Model: "gpt-4o-mini", CredentialMode: "MANAGED", Effort: "LOW"}
}

func TestGateCompleteManaged(t *testing.T) {
	chat := &stubChatCompleter{text: "hello", in: 3, out: 5}
	g := &AIGate{ManagedConfigured: true, ManagedKey: "sk-managed", Chat: chat}
	c, err := g.Complete(context.Background(), uuid.New(), managedAI(), "sys", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if c.Text != "hello" || c.InputTokens != 3 || c.OutputTokens != 5 {
		t.Fatalf("got %+v", c)
	}
	if chat.gotKey != "sk-managed" || chat.gotModel != "gpt-4o-mini" || chat.gotUser != "hi" || chat.gotSys != "sys" {
		t.Fatalf("call args: %+v", chat)
	}
}

func TestGateCompleteManagedNotConfigured(t *testing.T) {
	g := &AIGate{Chat: &stubChatCompleter{}}
	_, err := g.Complete(context.Background(), uuid.New(), managedAI(), "", "hi")
	if errCode(t, err) != domain.CodeNotConfigured {
		t.Fatalf("want NOT_CONFIGURED, got %v", err)
	}
}

func TestGateCompleteByokDecrypts(t *testing.T) {
	userID := uuid.New()
	chat := &stubChatCompleter{text: "ok"}
	cipher := &stubCipher{plaintext: "sk-user"}
	keys := &stubAiKeyStore{key: &domain.AiKey{UserID: userID, Provider: "OPENAI", Ciphertext: []byte{1}, Nonce: []byte{2}}}
	g := &AIGate{ByokEnabled: true, Keys: keys, Cipher: cipher, Chat: chat}
	ai := &domain.AiOptions{Provider: "OPENAI", Model: "gpt-4o", CredentialMode: "BYOK", Effort: "HIGH"}
	c, err := g.Complete(context.Background(), userID, ai, "", "yo")
	if err != nil {
		t.Fatal(err)
	}
	if c.Text != "ok" || chat.gotKey != "sk-user" {
		t.Fatalf("got %+v key=%q", c, chat.gotKey)
	}
}

func TestGateCompleteByokMissingKey(t *testing.T) {
	g := &AIGate{ByokEnabled: true, Keys: &stubAiKeyStore{}, Cipher: &stubCipher{}, Chat: &stubChatCompleter{}}
	ai := &domain.AiOptions{Provider: "OPENAI", Model: "gpt-4o", CredentialMode: "BYOK", Effort: "LOW"}
	_, err := g.Complete(context.Background(), uuid.New(), ai, "", "hi")
	if errCode(t, err) != domain.CodeIntegrationRequired {
		t.Fatalf("want INTEGRATION_REQUIRED, got %v", err)
	}
}

func TestGateCompleteUnsupportedProvider(t *testing.T) {
	userID := uuid.New()
	keys := &stubAiKeyStore{key: &domain.AiKey{UserID: userID, Provider: "CLAUDE"}}
	g := &AIGate{ByokEnabled: true, Keys: keys, Cipher: &stubCipher{plaintext: "k"}, Chat: &stubChatCompleter{}}
	ai := &domain.AiOptions{Provider: "CLAUDE", Model: "claude-sonnet-4", CredentialMode: "BYOK", Effort: "LOW"}
	_, err := g.Complete(context.Background(), userID, ai, "", "hi")
	if errCode(t, err) != domain.CodeNotConfigured {
		t.Fatalf("want NOT_CONFIGURED, got %v", err)
	}
}

func TestGateCompleteProviderFailure(t *testing.T) {
	chat := &stubChatCompleter{err: errors.New("boom")}
	g := &AIGate{ManagedConfigured: true, ManagedKey: "sk", Chat: chat}
	_, err := g.Complete(context.Background(), uuid.New(), managedAI(), "", "hi")
	if errCode(t, err) != domain.CodeProviderError {
		t.Fatalf("want PROVIDER_ERROR, got %v", err)
	}
}

func TestAIServiceGenerateCreatesOperationAndUsage(t *testing.T) {
	userID, appID := uuid.New(), uuid.New()
	ops := &stubOperationStore{}
	usage := &stubAiUsageStore{}
	chat := &stubChatCompleter{text: "generated", in: 10, out: 20}
	gate := &AIGate{ManagedConfigured: true, ManagedKey: "sk", Chat: chat}
	svc := NewAIService(gate, ops, usage, &stubApplicationStore{app: &domain.Application{ID: appID, UserID: userID}}, noopUoW{})
	op, err := svc.Generate(context.Background(), userID, AIGenerateInput{
		AI: managedAI(), Prompt: "write", ApplicationID: appID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if op.Type != domain.OpAIGenerate || op.Status != domain.OpSucceeded {
		t.Fatalf("op: %+v", op)
	}
	if len(usage.items) != 1 {
		t.Fatal("usage not recorded")
	}
	u := usage.items[0]
	if u.InputTokens != 10 || u.OutputTokens != 20 || !u.Managed || u.Provider != "OPENAI" || u.Status != domain.UsageSettled {
		t.Fatalf("usage: %+v", u)
	}
	if u.OperationID == nil || *u.OperationID != op.ID {
		t.Fatal("usage missing operationId")
	}
}

func TestAIServiceGenerateBadApplication(t *testing.T) {
	svc := NewAIService(&AIGate{}, &stubOperationStore{}, &stubAiUsageStore{}, &stubApplicationStore{}, noopUoW{})
	_, err := svc.Generate(context.Background(), uuid.New(), AIGenerateInput{
		AI: managedAI(), Prompt: "x", ApplicationID: uuid.New(),
	})
	if errCode(t, err) != domain.CodeNotFound {
		t.Fatalf("want NOT_FOUND, got %v", err)
	}
}

func TestAIServiceListUsage(t *testing.T) {
	userID := uuid.New()
	usage := &stubAiUsageStore{items: []domain.AiUsage{{UserID: userID}, {UserID: uuid.New()}}}
	svc := NewAIService(&AIGate{}, &stubOperationStore{}, usage, &stubApplicationStore{}, noopUoW{})
	p, err := svc.ListUsage(context.Background(), userID, domain.AiUsageFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 1 {
		t.Fatalf("want 1, got %d", len(p.Items))
	}
}

func TestPostMessageAICompletion(t *testing.T) {
	userID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1}
	convs := &stubConversationStore{conv: conv}
	usage := &stubAiUsageStore{}
	chat := &stubChatCompleter{text: "real answer", in: 7, out: 9}
	gate := &AIGate{ManagedConfigured: true, ManagedKey: "sk", Chat: chat}
	svc := NewConversationService(convs, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{}, &stubOperationStore{}, noopUoW{}, gate, usage)
	op, err := svc.PostMessage(context.Background(), userID, conv.ID, PostMessageInput{
		Text: "question", AccessMode: domain.AccessSuggest, Context: MessageContext{}, AI: managedAI(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(convs.messages) != 2 {
		t.Fatalf("want 2 messages, got %d", len(convs.messages))
	}
	if convs.messages[1].Text != "real answer" {
		t.Fatalf("assistant text: %q", convs.messages[1].Text)
	}
	if len(usage.items) != 1 || usage.items[0].OperationID == nil || *usage.items[0].OperationID != op.ID {
		t.Fatalf("usage: %+v", usage.items)
	}
}

func TestPostMessageNoAIKeepsStub(t *testing.T) {
	userID := uuid.New()
	conv := &domain.Conversation{ID: uuid.New(), UserID: userID, Revision: 1}
	convs := &stubConversationStore{conv: conv}
	chat := &stubChatCompleter{text: "should not be used"}
	gate := &AIGate{ManagedConfigured: true, ManagedKey: "sk", Chat: chat}
	svc := NewConversationService(convs, &stubApplicationStore{}, &stubDocumentStore{}, &stubEvidenceStore{}, &stubOperationStore{}, noopUoW{}, gate, &stubAiUsageStore{})
	_, err := svc.PostMessage(context.Background(), userID, conv.ID, PostMessageInput{
		Text: "hi", AccessMode: domain.AccessSuggest, Context: MessageContext{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if convs.messages[1].Text == "should not be used" {
		t.Fatal("stub reply expected")
	}
}
