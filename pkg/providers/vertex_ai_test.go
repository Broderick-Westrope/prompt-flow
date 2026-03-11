package providers

import (
	"testing"
)

func TestVertexAIProvider_Name(t *testing.T) {
	p := NewVertexAIProvider("test-project", "")
	if got := p.Name(); got != "vertex_ai" {
		t.Errorf("Name() = %q, want %q", got, "vertex_ai")
	}
}

func TestNewVertexAIProvider_DefaultLocation(t *testing.T) {
	p := NewVertexAIProvider("test-project", "")
	if p.location != "us-central1" {
		t.Errorf("location = %q, want %q", p.location, "us-central1")
	}
}

func TestNewVertexAIProvider_CustomLocation(t *testing.T) {
	p := NewVertexAIProvider("test-project", "europe-west1")
	if p.location != "europe-west1" {
		t.Errorf("location = %q, want %q", p.location, "europe-west1")
	}
}

func TestNewVertexAIProvider_StoresProjectID(t *testing.T) {
	p := NewVertexAIProvider("my-project", "")
	if p.projectID != "my-project" {
		t.Errorf("projectID = %q, want %q", p.projectID, "my-project")
	}
}

func TestEstimateVertexAICost(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		wantInput    float64
		wantOutput   float64
	}{
		{
			name:         "gemini-2.0-flash",
			model:        "gemini-2.0-flash",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantInput:    0.10,
			wantOutput:   0.40,
		},
		{
			name:         "gemini-2.5-pro",
			model:        "gemini-2.5-pro",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantInput:    1.25,
			wantOutput:   10.0,
		},
		{
			name:         "gemini-2.5-flash",
			model:        "gemini-2.5-flash",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantInput:    0.15,
			wantOutput:   0.60,
		},
		{
			name:         "gemini-2.0-flash-lite",
			model:        "gemini-2.0-flash-lite",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantInput:    0.075,
			wantOutput:   0.30,
		},
		{
			name:         "unknown model uses flash pricing",
			model:        "unknown-model",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantInput:    0.10,
			wantOutput:   0.40,
		},
		{
			name:         "zero tokens",
			model:        "gemini-2.0-flash",
			inputTokens:  0,
			outputTokens: 0,
			wantInput:    0,
			wantOutput:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInput, gotOutput := estimateVertexAICost(tt.model, tt.inputTokens, tt.outputTokens)
			if gotInput != tt.wantInput {
				t.Errorf("input cost = %v, want %v", gotInput, tt.wantInput)
			}
			if gotOutput != tt.wantOutput {
				t.Errorf("output cost = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}
