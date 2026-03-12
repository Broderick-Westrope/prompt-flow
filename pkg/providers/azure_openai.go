package providers

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// AzureOpenAIProvider implements the Provider interface for Azure OpenAI Service.
// Unlike the standard OpenAI provider, Azure OpenAI requires an endpoint URL and
// the "model" field in flow YAML maps to the Azure deployment name, not the OpenAI
// model name (e.g., if your deployment is named "my-gpt4o", use model: "my-gpt4o").
type AzureOpenAIProvider struct {
	client    *openai.Client
	endpoint  string
	apiKeySet bool
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider.
// The endpoint is the Azure OpenAI resource URL (e.g., "https://your-resource.openai.azure.com/").
// The apiKey is the Azure OpenAI API key.
func NewAzureOpenAIProvider(endpoint, apiKey string) *AzureOpenAIProvider {
	config := openai.DefaultAzureConfig(apiKey, endpoint)
	client := openai.NewClientWithConfig(config)

	return &AzureOpenAIProvider{
		client:    client,
		endpoint:  endpoint,
		apiKeySet: apiKey != "",
	}
}

func (p *AzureOpenAIProvider) Name() string {
	return "azure_openai"
}

func (p *AzureOpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.apiKeySet {
		return nil, fmt.Errorf("azure OpenAI provider received an empty API key")
	}

	if p.endpoint == "" {
		return nil, fmt.Errorf("azure OpenAI provider received an empty endpoint (set AZURE_OPENAI_ENDPOINT)")
	}

	// Build the request — identical to OpenAI, but req.Model maps to the Azure deployment name
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Prompt,
		},
	}

	temperature := 0.7
	maxTokens := 1000

	if temp, ok := req.Settings["temperature"].(float64); ok {
		temperature = temp
	}
	if max, ok := req.Settings["max_tokens"].(float64); ok {
		maxTokens = int(max)
	} else if max, ok := req.Settings["max_tokens"].(int); ok {
		maxTokens = max
	}

	chatReq := openai.ChatCompletionRequest{
		Model:       req.Model, // Maps to Azure deployment name, not OpenAI model name
		Messages:    messages,
		Temperature: float32(temperature),
		MaxTokens:   maxTokens,
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("azure OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Azure OpenAI")
	}

	inputCost, outputCost := estimateAzureOpenAICost(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return &CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		Model:        resp.Model,
	}, nil
}

// estimateAzureOpenAICost calculates approximate cost for Azure OpenAI.
// Azure OpenAI pricing varies by region, tier (standard vs provisioned), and commitment.
// These are rough estimates using standard pay-as-you-go rates. Actual costs depend on
// your Azure pricing tier and region. The model name is the Azure deployment name, not
// the OpenAI model name, so we use a generic default rate.
// Pricing as of March 2026 (USD per 1M tokens).
func estimateAzureOpenAICost(inputTokens, outputTokens int) (float64, float64) {
	// Default to approximate GPT-4o-class pricing as a reasonable middle ground.
	// Azure deployment names don't map to known models, so per-model pricing isn't feasible.
	inputRate := 2.50  // USD per 1M input tokens
	outputRate := 10.0 // USD per 1M output tokens

	inputCost := (float64(inputTokens) / 1_000_000.0) * inputRate
	outputCost := (float64(outputTokens) / 1_000_000.0) * outputRate

	return inputCost, outputCost
}
