package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AICompletion struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

type ChatCompleter interface {
	Chat(ctx context.Context, apiKey, model, system, user string) (*AICompletion, error)
}

type KeyCipher interface {
	Decrypt(ciphertext, nonce []byte, userID uuid.UUID, provider string) (string, error)
}

type AiUsageStatus string

const (
	UsageReserved AiUsageStatus = "RESERVED"
	UsageSettled  AiUsageStatus = "SETTLED"
	UsageReleased AiUsageStatus = "RELEASED"
)

type AiUsage struct {
	ID               uuid.UUID     `json:"id"`
	UserID           uuid.UUID     `json:"-"`
	OperationID      *uuid.UUID    `json:"operationId"`
	Provider         string        `json:"provider"`
	Model            string        `json:"model"`
	Managed          bool          `json:"managed"`
	InputTokens      int           `json:"inputTokens"`
	OutputTokens     int           `json:"outputTokens"`
	CostMicroCredits int64         `json:"costMicroCredits"`
	Status           AiUsageStatus `json:"status"`
	CreatedAt        time.Time     `json:"createdAt"`
}

type AiUsageFilter struct {
	From *time.Time
	To   *time.Time
	Page PageRequest
}

type AiUsageStore interface {
	Create(ctx context.Context, u *AiUsage) error
	List(ctx context.Context, userID uuid.UUID, f AiUsageFilter, limit int) ([]AiUsage, error)
}

type AIModel struct {
	Provider string
	Model    string
	Label    string
}

var OpenAIModels = []AIModel{
	{Provider: "OPENAI", Model: "gpt-4o-mini", Label: "GPT-4o mini"},
	{Provider: "OPENAI", Model: "gpt-4o", Label: "GPT-4o"},
}

var openAIMicroPer1M = map[string][2]int64{
	"gpt-4o-mini": {150000, 600000},
	"gpt-4o":      {2500000, 10000000},
}

func AICostMicroCredits(provider, model string, inputTokens, outputTokens int) int64 {
	if provider != "OPENAI" {
		return 0
	}
	rates, ok := openAIMicroPer1M[model]
	if !ok {
		rates = openAIMicroPer1M["gpt-4o-mini"]
	}
	return (int64(inputTokens)*rates[0] + int64(outputTokens)*rates[1]) / 1_000_000
}
