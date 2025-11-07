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
	inputCost, outputCost := estimateOpenAICost(req.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return &CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		Model:        resp.Model,
	}, nil
}

// estimateOpenAICost calculates approximate cost based on model and token usage.
// Does not consider cached input or other edge cases; only standard input and output is considered.
// Pricing as of 07/11/2025 (USD per 1M tokens).
// https://platform.openai.com/docs/pricing?latest-pricing=standard
func estimateOpenAICost(model string, inputTokens, outputTokens int) (float64, float64) {
	var inputCost, outputCost float64

	switch model {
	case "gpt-5":
		inputCost = 1.25
		outputCost = 10.0
	case "gpt-5-mini":
		inputCost = 0.25
		outputCost = 2.0
	case "gpt-5-nano":
		inputCost = 0.05
		outputCost = 0.4

	case "gpt-4o-mini":
		inputCost = 0.15
		outputCost = 0.60
	}

	inputCost *= (float64(inputTokens) / 1_000_000.0)
	outputCost *= (float64(outputTokens) / 1_000_000.0)

	return inputCost, outputCost
}
