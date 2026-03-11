package providers

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/vertexai/genai"
)

// VertexAIProvider implements the Provider interface for Google Vertex AI (Gemini models)
type VertexAIProvider struct {
	projectID string
	location  string

	client     *genai.Client
	clientOnce sync.Once
	clientErr  error
}

// NewVertexAIProvider creates a new Vertex AI provider.
// It uses Application Default Credentials (ADC) for authentication.
// On GCP (Cloud Run, GKE), ADC is automatic. Locally, run:
//
//	gcloud auth application-default login
func NewVertexAIProvider(projectID, location string) *VertexAIProvider {
	if location == "" {
		location = "us-central1"
	}
	return &VertexAIProvider{
		projectID: projectID,
		location:  location,
	}
}

// Name returns the provider name
func (p *VertexAIProvider) Name() string {
	return "vertex_ai"
}

// getClient returns a cached genai.Client, creating it on first use.
func (p *VertexAIProvider) getClient(ctx context.Context) (*genai.Client, error) {
	p.clientOnce.Do(func() {
		p.client, p.clientErr = genai.NewClient(ctx, p.projectID, p.location)
	})
	if p.clientErr != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client (check ADC: gcloud auth application-default login): %w", p.clientErr)
	}
	return p.client, nil
}

// Complete sends a prompt to Vertex AI and returns the response
func (p *VertexAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	client, err := p.getClient(ctx)
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel(req.Model)

	// Apply settings with defaults matching other providers
	temperature := float32(0.7)
	maxTokens := int32(1000)

	if temp, ok := req.Settings["temperature"].(float64); ok {
		temperature = float32(temp)
	}
	model.SetTemperature(temperature)

	if mt, ok := req.Settings["max_tokens"].(float64); ok {
		maxTokens = int32(mt)
	} else if mt, ok := req.Settings["max_tokens"].(int); ok {
		maxTokens = int32(mt)
	}
	model.SetMaxOutputTokens(maxTokens)

	if topP, ok := req.Settings["top_p"].(float64); ok {
		tp := float32(topP)
		model.TopP = &tp
	}

	resp, err := model.GenerateContent(ctx, genai.Text(req.Prompt))
	if err != nil {
		return nil, fmt.Errorf("Vertex AI API call failed: %w", err)
	}

	if len(resp.Candidates) == 0 ||
		resp.Candidates[0].Content == nil ||
		len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Vertex AI")
	}

	// Extract text from response using type assertion
	textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response part type from Vertex AI: %T", resp.Candidates[0].Content.Parts[0])
	}
	content := string(textPart)

	// Extract token usage
	var inputTokens, outputTokens int
	if resp.UsageMetadata != nil {
		inputTokens = int(resp.UsageMetadata.PromptTokenCount)
		outputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}

	inputCost, outputCost := estimateVertexAICost(req.Model, inputTokens, outputTokens)

	return &CompletionResponse{
		Content:      content,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		Model:        req.Model,
	}, nil
}

// estimateVertexAICost calculates approximate cost based on model and token usage.
// Pricing as of 11/03/2026 (USD per 1M tokens).
// https://cloud.google.com/vertex-ai/generative-ai/pricing
func estimateVertexAICost(model string, inputTokens, outputTokens int) (float64, float64) {
	var inputRate, outputRate float64

	switch model {
	case "gemini-2.5-pro":
		inputRate = 1.25
		outputRate = 10.0
	case "gemini-2.5-flash":
		inputRate = 0.15
		outputRate = 0.60
	case "gemini-2.0-flash":
		inputRate = 0.10
		outputRate = 0.40
	case "gemini-2.0-flash-lite":
		inputRate = 0.075
		outputRate = 0.30
	default:
		// Default to flash pricing
		inputRate = 0.10
		outputRate = 0.40
	}

	inputCost := (float64(inputTokens) / 1_000_000.0) * inputRate
	outputCost := (float64(outputTokens) / 1_000_000.0) * outputRate

	return inputCost, outputCost
}
