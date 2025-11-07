package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// GithubPlaygroundOpenAIProvider implements the Provider interface for Github Playground as an OpenAI provider
type GithubPlaygroundOpenAIProvider struct {
	client    *openai.Client
	apiKeySet bool
}

// NewGithubPlaygroundProvider creates a new Github Playground provider
func NewGithubPlaygroundOpenAIProvider(apiKey string) *GithubPlaygroundOpenAIProvider {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = "https://models.github.ai/inference"
	client := openai.NewClientWithConfig(cfg)

	return &GithubPlaygroundOpenAIProvider{
		client:    client,
		apiKeySet: apiKey != "",
	}
}

// Name returns the provider name
func (p *GithubPlaygroundOpenAIProvider) Name() string {
	return "github_playground_openai"
}

// Complete sends a prompt to Github Playground and returns the response
func (p *GithubPlaygroundOpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.apiKeySet {
		return nil, fmt.Errorf("GitHub Playground OpenAI Provider received an empty API key")
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
		return nil, fmt.Errorf("GitHub Playground OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from GitHub Playground OpenAI")
	}

	// Calculate estimated cost (approximate pricing)
	inputCost, outputCost := estimateOpenAICost(strings.TrimPrefix(req.Model, "openai/"), resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return &CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		Model:        resp.Model,
	}, nil
}
