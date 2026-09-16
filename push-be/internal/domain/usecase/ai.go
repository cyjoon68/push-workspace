package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type AIService struct {
	gate         *AIGate
	ops          domain.OperationStore
	usage        domain.AiUsageStore
	applications domain.ApplicationStore
	uow          domain.UnitOfWork
}

func NewAIService(gate *AIGate, ops domain.OperationStore, usage domain.AiUsageStore, applications domain.ApplicationStore, uow domain.UnitOfWork) *AIService {
	return &AIService{gate: gate, ops: ops, usage: usage, applications: applications, uow: uow}
}

type AIGenerateInput struct {
	AI            *domain.AiOptions
	Prompt        string
	ApplicationID uuid.UUID
	EvidenceIDs   []uuid.UUID
}

func RecordUsage(ctx context.Context, store domain.AiUsageStore, userID uuid.UUID, opID uuid.UUID, ai *domain.AiOptions, c *domain.AICompletion) error {
	return store.Create(ctx, &domain.AiUsage{
		ID: uuid.New(), UserID: userID, OperationID: &opID,
		Provider: ai.Provider, Model: ai.Model,
		Managed:          ai.CredentialMode == "MANAGED",
		InputTokens:      c.InputTokens,
		OutputTokens:     c.OutputTokens,
		CostMicroCredits: domain.AICostMicroCredits(ai.Provider, ai.Model, c.InputTokens, c.OutputTokens),
		Status:           domain.UsageSettled,
		CreatedAt:        time.Now().UTC(),
	})
}

func (s *AIService) Generate(ctx context.Context, userID uuid.UUID, in AIGenerateInput) (*domain.Operation, error) {
	if in.AI == nil {
		return nil, domain.ValidationField("ai", "required")
	}
	if in.Prompt == "" || domain.CodePointLen(in.Prompt) > 20000 {
		return nil, domain.ValidationField("prompt", "prompt must be 1-20000 characters")
	}
	if _, err := s.applications.Get(ctx, userID, in.ApplicationID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	c, derr := s.gate.Complete(ctx, userID, in.AI, "", in.Prompt)
	if derr != nil {
		return nil, derr
	}
	var op *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		now := time.Now().UTC()
		opID := uuid.New()
		res, _ := json.Marshal(map[string]any{
			"text":      c.Text,
			"citations": []any{},
			"usage": map[string]any{
				"inputTokens": c.InputTokens, "outputTokens": c.OutputTokens,
			},
		})
		op = &domain.Operation{
			ID: opID, UserID: userID, Type: domain.OpAIGenerate,
			ApplicationID: &in.ApplicationID, Status: domain.OpSucceeded,
			Result:    &domain.OperationResult{Kind: string(domain.OpAIGenerate), Value: res},
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.ops.Create(ctx, op); err != nil {
			return err
		}
		return RecordUsage(ctx, s.usage, userID, opID, in.AI, c)
	})
	if err != nil {
		return nil, domain.Internal()
	}
	return op, nil
}

func (s *AIService) ListUsage(ctx context.Context, userID uuid.UUID, f domain.AiUsageFilter) (domain.Page[domain.AiUsage], error) {
	limit := f.Page.EffectiveLimit()
	items, err := s.usage.List(ctx, userID, f, limit+1)
	if err != nil {
		return domain.Page[domain.AiUsage]{}, domain.Internal()
	}
	return domain.NewPage(items, limit, func(u domain.AiUsage) domain.Cursor {
		return domain.Cursor{CreatedAt: u.CreatedAt, ID: u.ID}
	}), nil
}
