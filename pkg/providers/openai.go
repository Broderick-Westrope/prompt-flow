package providers

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	client    *openai.Client
	apiKeySet bool
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	client := openai.NewClient(apiKey)

	return &OpenAIProvider{
		client:    client,
		apiKeySet: apiKey != "",
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Complete sends a prompt to OpenAI and returns the response
func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.apiKeySet {
		return nil, fmt.Errorf("OpenAI Provider received an empty API key")
	}

	// Build the request
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Prompt,
		},
	}

	// Get settings with defaults
	temperature := 0.7
	maxTokens := 1000

	if temp, ok := req.Settings["temperature"].(float64); ok {
		temperature = temp
	}
	if max, ok := req.Settings["max_tokens"].(int); ok {
		maxTokens = max
	}

	chatReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: float32(temperature),
		MaxTokens:   maxTokens,
	}

	// Call OpenAI
	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	// Calculate estimated cost (approximate pricing)
	cost := estimateOpenAICost(req.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return &CompletionResponse{
		Content:          resp.Choices[0].Message.Content,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		EstimatedCost:    cost,
		Model:            resp.Model,
	}, nil
}

// estimateOpenAICost calculates approximate cost based on model and token usage
func estimateOpenAICost(model string, promptTokens, completionTokens int) float64 {
	// Pricing as of 2024 (per 1M tokens)
	// These are approximate and should be updated
	var promptCost, completionCost float64

	switch model {
	case "gpt-4", "gpt-4-0613":
		promptCost = 30.0     // $30 per 1M tokens
		completionCost = 60.0 // $60 per 1M tokens
	case "gpt-4-turbo", "gpt-4-turbo-preview", "gpt-4-1106-preview":
		promptCost = 10.0     // $10 per 1M tokens
		completionCost = 30.0 // $30 per 1M tokens
	case "gpt-3.5-turbo", "gpt-3.5-turbo-0125":
		promptCost = 0.5     // $0.50 per 1M tokens
		completionCost = 1.5 // $1.50 per 1M tokens
	default:
		// Default to GPT-3.5 pricing
		promptCost = 0.5
		completionCost = 1.5
	}

	promptCostUSD := (float64(promptTokens) / 1_000_000.0) * promptCost
	completionCostUSD := (float64(completionTokens) / 1_000_000.0) * completionCost

	return promptCostUSD + completionCostUSD
}
