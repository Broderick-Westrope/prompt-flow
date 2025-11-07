package providers

import (
	"context"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
)

// AnthropicProvider implements the Provider interface for Anthropic
type AnthropicProvider struct {
	client    *anthropic.Client
	apiKeySet bool
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	client := anthropic.NewClient(apiKey)

	return &AnthropicProvider{
		client:    client,
		apiKeySet: apiKey != "",
	}
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Complete sends a prompt to Anthropic and returns the response
func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.apiKeySet {
		return nil, fmt.Errorf("Anthropic API key not set (use ANTHROPIC_API_KEY environment variable)")
	}

	// Get settings with defaults
	temperature := float32(0.7)
	maxTokens := 1000

	if temp, ok := req.Settings["temperature"].(float64); ok {
		temperature = float32(temp)
	}
	if max, ok := req.Settings["max_tokens"].(int); ok {
		maxTokens = max
	}

	// Build the request
	messages := []anthropic.Message{
		anthropic.NewUserTextMessage(req.Prompt),
	}

	chatReq := anthropic.MessagesRequest{
		Model:       anthropic.Model(req.Model),
		Messages:    messages,
		Temperature: &temperature,
		MaxTokens:   maxTokens,
	}

	// Call Anthropic
	resp, err := p.client.CreateMessages(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API call failed: %w", err)
	}

	// Extract text content
	content := resp.GetFirstContentText()
	if content == "" {
		return nil, fmt.Errorf("no text content in response from Anthropic")
	}

	// Calculate estimated cost
	cost := estimateAnthropicCost(req.Model, resp.Usage.InputTokens, resp.Usage.OutputTokens)

	return &CompletionResponse{
		Content:          content,
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		EstimatedCost:    cost,
		Model:            string(resp.Model),
	}, nil
}

// estimateAnthropicCost calculates approximate cost based on model and token usage
func estimateAnthropicCost(model string, inputTokens, outputTokens int) float64 {
	// Pricing as of 2024 (per 1M tokens)
	var inputCost, outputCost float64

	switch model {
	case "claude-3-opus-20240229":
		inputCost = 15.0  // $15 per 1M tokens
		outputCost = 75.0 // $75 per 1M tokens
	case "claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20240620":
		inputCost = 3.0   // $3 per 1M tokens
		outputCost = 15.0 // $15 per 1M tokens
	case "claude-3-sonnet-20240229":
		inputCost = 3.0   // $3 per 1M tokens
		outputCost = 15.0 // $15 per 1M tokens
	case "claude-3-haiku-20240307":
		inputCost = 0.25  // $0.25 per 1M tokens
		outputCost = 1.25 // $1.25 per 1M tokens
	default:
		// Default to Haiku pricing
		inputCost = 0.25
		outputCost = 1.25
	}

	inputCostUSD := (float64(inputTokens) / 1_000_000.0) * inputCost
	outputCostUSD := (float64(outputTokens) / 1_000_000.0) * outputCost

	return inputCostUSD + outputCostUSD
}
