package providers

import (
	"context"
	"os"
)

// Provider is the interface that all LLM providers must implement
type Provider interface {
	// Name returns the provider name (e.g., "openai", "anthropic")
	Name() string

	// Complete sends a prompt to the LLM and returns the response
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

// CompletionRequest represents a request to an LLM
type CompletionRequest struct {
	Prompt   string         // The prompt text
	Model    string         // Model identifier
	Settings map[string]any // Provider-specific settings
}

// CompletionResponse represents a response from an LLM
type CompletionResponse struct {
	Content      string  // The generated text
	InputTokens  int     // Tokens in the prompt
	OutputTokens int     // Tokens in the completion
	InputCost    float64 // Cost in USD for the input tokens
	OutputCost   float64 // Cost in USD for the output tokens
	Model        string  // Model that was used
}

// Registry holds all available providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// WithDefaultProviders registers the default providers with the following environment variables:
//  1. OpenAI: OPENAI_API_KEY
//  2. Anthropic: ANTHROPIC_API_KEY
//  3. Github Playground OpenAI: GITHUB_PLAYGROUND_PAT
//  4. Vertex AI: GCP_PROJECT_ID (optional: GCP_LOCATION, defaults to us-central1)
//  5. Azure OpenAI: AZURE_OPENAI_ENDPOINT (optional: AZURE_OPENAI_API_KEY)
func (r *Registry) WithDefaultProviders() *Registry {
	r.Register(NewOpenAIProvider(os.Getenv("OPENAI_API_KEY")))
	r.Register(NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY")))
	r.Register(NewGithubPlaygroundOpenAIProvider(os.Getenv("GITHUB_PLAYGROUND_PAT")))
	if projectID := os.Getenv("GCP_PROJECT_ID"); projectID != "" {
		r.Register(NewVertexAIProvider(projectID, os.Getenv("GCP_LOCATION")))
	}
	if endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT"); endpoint != "" {
		r.Register(NewAzureOpenAIProvider(endpoint, os.Getenv("AZURE_OPENAI_API_KEY")))
	}
	return r
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered provider names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
