package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"push-be/internal/domain"
)

type OpenAIClient struct {
	HTTP    *http.Client
	BaseURL string
}

func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		HTTP:    &http.Client{Timeout: 60 * time.Second},
		BaseURL: "https://api.openai.com/v1",
	}
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *OpenAIClient) Chat(ctx context.Context, apiKey, model, system, user string) (*domain.AICompletion, error) {
	messages := []openAIChatMessage{}
	if system != "" {
		messages = append(messages, openAIChatMessage{Role: "system", Content: system})
	}
	messages = append(messages, openAIChatMessage{Role: "user", Content: user})
	body, err := json.Marshal(openAIChatRequest{Model: model, Messages: messages})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	res, err := o.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var out openAIChatResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("openai status %d", res.StatusCode)
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return nil, fmt.Errorf("%s", msg)
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}
	return &domain.AICompletion{
		Text:         out.Choices[0].Message.Content,
		InputTokens:  out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
	}, nil
}
