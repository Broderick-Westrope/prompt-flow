package providers

import (
	"context"
	"testing"
)

func TestAzureOpenAIProvider_Name(t *testing.T) {
	p := NewAzureOpenAIProvider("https://test.openai.azure.com/", "test-key")
	if got := p.Name(); got != "azure_openai" {
		t.Errorf("Name() = %q, want %q", got, "azure_openai")
	}
}

func TestNewAzureOpenAIProvider_StoresEndpoint(t *testing.T) {
	p := NewAzureOpenAIProvider("https://test.openai.azure.com/", "test-key")
	if p.endpoint != "https://test.openai.azure.com/" {
		t.Errorf("endpoint = %q, want %q", p.endpoint, "https://test.openai.azure.com/")
	}
}

func TestNewAzureOpenAIProvider_APIKeySet(t *testing.T) {
	p := NewAzureOpenAIProvider("https://test.openai.azure.com/", "test-key")
	if !p.apiKeySet {
		t.Error("apiKeySet = false, want true")
	}
}

func TestNewAzureOpenAIProvider_EmptyAPIKey(t *testing.T) {
	p := NewAzureOpenAIProvider("https://test.openai.azure.com/", "")
	if p.apiKeySet {
		t.Error("apiKeySet = true, want false")
	}
}

func TestAzureOpenAIProvider_Complete_EmptyAPIKey(t *testing.T) {
	p := NewAzureOpenAIProvider("https://test.openai.azure.com/", "")
	_, err := p.Complete(context.Background(), CompletionRequest{
		Prompt: "test",
		Model:  "my-deployment",
	})
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if got := err.Error(); got != "azure OpenAI provider received an empty API key" {
		t.Errorf("error = %q, want descriptive API key error", got)
	}
}

func TestAzureOpenAIProvider_Complete_EmptyEndpoint(t *testing.T) {
	p := NewAzureOpenAIProvider("", "test-key")
	_, err := p.Complete(context.Background(), CompletionRequest{
		Prompt: "test",
		Model:  "my-deployment",
	})
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
	if got := err.Error(); got != "azure OpenAI provider received an empty endpoint (set AZURE_OPENAI_ENDPOINT)" {
		t.Errorf("error = %q, want descriptive endpoint error", got)
	}
}

func TestEstimateAzureOpenAICost(t *testing.T) {
	tests := []struct {
		name         string
		inputTokens  int
		outputTokens int
		wantInput    float64
		wantOutput   float64
	}{
		{
			name:         "1M tokens",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantInput:    2.50,
			wantOutput:   10.0,
		},
		{
			name:         "zero tokens",
			inputTokens:  0,
			outputTokens: 0,
			wantInput:    0,
			wantOutput:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInput, gotOutput := estimateAzureOpenAICost(tt.inputTokens, tt.outputTokens)
			if gotInput != tt.wantInput {
				t.Errorf("input cost = %v, want %v", gotInput, tt.wantInput)
			}
			if gotOutput != tt.wantOutput {
				t.Errorf("output cost = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}
