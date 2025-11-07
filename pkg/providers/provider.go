package providers

import (
	"context"
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
