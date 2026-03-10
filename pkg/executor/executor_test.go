package executor

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/broderick/prompt-flow/pkg/providers"
)

// mockProvider implements providers.Provider for testing.
type mockProvider struct {
	name      string
	responses map[string]string             // prompt content -> response content
	calls     []providers.CompletionRequest // recorded calls for assertions
	defaultR  string                        // fallback response if no match
	err       error                         // if set, Complete returns this error
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:      name,
		responses: make(map[string]string),
		defaultR:  "default response",
	}
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Complete(_ context.Context, req providers.CompletionRequest) (*providers.CompletionResponse, error) {
	m.calls = append(m.calls, req)

	if m.err != nil {
		return nil, m.err
	}

	content, ok := m.responses[req.Prompt]
	if !ok {
		content = m.defaultR
	}

	return &providers.CompletionResponse{
		Content:      content,
		InputTokens:  10,
		OutputTokens: 5,
		InputCost:    0.001,
		OutputCost:   0.0005,
		Model:        req.Model,
	}, nil
}

// helper to build a minimal valid flow with defaults filled in.
func baseFlow(nodes []flow.Node) *flow.Flow {
	return &flow.Flow{
		Version: "1.0",
		Name:    "test-flow",
		Config: flow.Config{
			DefaultProvider: "mock",
			DefaultModel:    "test-model",
		},
		Nodes: nodes,
	}
}

// --- Topological sort tests (verified via execution order) ---

func TestTopologicalSort_LinearChain(t *testing.T) {
	// A → B → C
	mock := newMockProvider("mock")
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "A",
			Prompt:  "prompt A",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "B",
			Prompt:  "prompt B {{.a_out}}",
			Inputs:  []flow.Input{{Name: "a_out", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "C",
			Prompt:  "prompt C {{.b_out}}",
			Inputs:  []flow.Input{{Name: "b_out", From: "B.result"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"x": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify execution order via NodeResults
	if len(result.NodeResults) != 3 {
		t.Fatalf("expected 3 node results, got %d", len(result.NodeResults))
	}

	order := []string{result.NodeResults[0].NodeID, result.NodeResults[1].NodeID, result.NodeResults[2].NodeID}
	// A must come before B, B must come before C
	idxA, idxB, idxC := indexOf(order, "A"), indexOf(order, "B"), indexOf(order, "C")
	if idxA >= idxB || idxB >= idxC {
		t.Errorf("expected order A < B < C, got %v", order)
	}
}

func TestTopologicalSort_FanOut(t *testing.T) {
	// A → B, A → C
	mock := newMockProvider("mock")
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "A",
			Prompt:  "prompt A",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "B",
			Prompt:  "prompt B {{.a_out}}",
			Inputs:  []flow.Input{{Name: "a_out", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "C",
			Prompt:  "prompt C {{.a_out}}",
			Inputs:  []flow.Input{{Name: "a_out", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"x": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order := nodeResultIDs(result)
	idxA := indexOf(order, "A")
	idxB := indexOf(order, "B")
	idxC := indexOf(order, "C")
	if idxA >= idxB || idxA >= idxC {
		t.Errorf("A must execute before B and C, got order: %v", order)
	}
}

func TestTopologicalSort_FanIn(t *testing.T) {
	// A → C, B → C (A and B are independent)
	mock := newMockProvider("mock")
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "A",
			Prompt:  "prompt A",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "B",
			Prompt:  "prompt B",
			Inputs:  []flow.Input{{Name: "y", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "C",
			Prompt:  "prompt C {{.a_out}} {{.b_out}}",
			Inputs: []flow.Input{
				{Name: "a_out", From: "A.result"},
				{Name: "b_out", From: "B.result"},
			},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"x": "val", "y": "val2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order := nodeResultIDs(result)
	idxA := indexOf(order, "A")
	idxB := indexOf(order, "B")
	idxC := indexOf(order, "C")
	if idxC <= idxA || idxC <= idxB {
		t.Errorf("C must execute after both A and B, got order: %v", order)
	}
}

func TestTopologicalSort_Diamond(t *testing.T) {
	// A → B, A → C, B → D, C → D
	mock := newMockProvider("mock")
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "A",
			Prompt:  "prompt A",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "B",
			Prompt:  "prompt B {{.a_out}}",
			Inputs:  []flow.Input{{Name: "a_out", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "C",
			Prompt:  "prompt C {{.a_out}}",
			Inputs:  []flow.Input{{Name: "a_out", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:     "D",
			Prompt: "prompt D {{.b_out}} {{.c_out}}",
			Inputs: []flow.Input{
				{Name: "b_out", From: "B.result"},
				{Name: "c_out", From: "C.result"},
			},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"x": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order := nodeResultIDs(result)
	idxA := indexOf(order, "A")
	idxB := indexOf(order, "B")
	idxC := indexOf(order, "C")
	idxD := indexOf(order, "D")

	if idxA >= idxB || idxA >= idxC {
		t.Errorf("A must execute before B and C, got order: %v", order)
	}
	if idxD <= idxB || idxD <= idxC {
		t.Errorf("D must execute after B and C, got order: %v", order)
	}
}

// --- Full execution tests ---

func TestExecute_SingleNode(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Hello world"] = "Hi there!"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "greeter",
			Prompt:  "Hello {{.user_input}}",
			Inputs:  []flow.Input{{Name: "user_input", From: "input"}},
			Outputs: []flow.Output{{Name: "greeting", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"user_input": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify the mock was called with the rendered prompt
	calls := mock.calls
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Prompt != "Hello world" {
		t.Errorf("expected prompt 'Hello world', got %q", calls[0].Prompt)
	}
	if calls[0].Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", calls[0].Model)
	}

	// Verify the flow output
	if result.Outputs["greeting"] != "Hi there!" {
		t.Errorf("expected output 'Hi there!', got %q", result.Outputs["greeting"])
	}
}

func TestExecute_MultiNodeChain(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Summarize: hello"] = "summary of hello"
	mock.responses["Translate: summary of hello"] = "translated summary"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "summarizer",
			Prompt:  "Summarize: {{.text}}",
			Inputs:  []flow.Input{{Name: "text", From: "input"}},
			Outputs: []flow.Output{{Name: "summary", To: ""}},
		},
		{
			ID:      "translator",
			Prompt:  "Translate: {{.input_text}}",
			Inputs:  []flow.Input{{Name: "input_text", From: "summarizer.summary"}},
			Outputs: []flow.Output{{Name: "translation", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Both nodes should have executed
	calls := mock.calls
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}

	if calls[0].Prompt != "Summarize: hello" {
		t.Errorf("expected first prompt 'Summarize: hello', got %q", calls[0].Prompt)
	}
	if calls[1].Prompt != "Translate: summary of hello" {
		t.Errorf("expected second prompt 'Translate: summary of hello', got %q", calls[1].Prompt)
	}

	// Final output
	if result.Outputs["translation"] != "translated summary" {
		t.Errorf("expected output 'translated summary', got %q", result.Outputs["translation"])
	}
}

// --- Error case tests ---

func TestExecute_MissingFlowInput(t *testing.T) {
	mock := newMockProvider("mock")
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "node1",
			Prompt:  "Hello {{.foo}}",
			Inputs:  []flow.Input{{Name: "foo", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	// Do not provide the "foo" input
	result, err := exec.Execute(context.Background(), f, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing flow input, got nil")
	}

	if result != nil && result.Success {
		t.Error("expected Success=false")
	}

	// No calls should have been made to the provider
	if len(mock.calls) != 0 {
		t.Errorf("expected 0 provider calls, got %d", len(mock.calls))
	}
}

func TestExecute_MissingProvider(t *testing.T) {
	// Registry with no providers registered
	registry := providers.NewRegistry()
	exec := New(registry)

	f := &flow.Flow{
		Version: "1.0",
		Name:    "test-flow",
		Config: flow.Config{
			DefaultProvider: "openai",
			DefaultModel:    "gpt-4",
		},
		Nodes: []flow.Node{
			{
				ID:      "node1",
				Prompt:  "Hello",
				Inputs:  []flow.Input{{Name: "x", From: "input"}},
				Outputs: []flow.Output{{Name: "result", To: "output"}},
			},
		},
	}

	_, err := exec.Execute(context.Background(), f, map[string]any{"x": "val"})
	if err == nil {
		t.Fatal("expected error for missing provider, got nil")
	}

	expected := "provider not found"
	if got := fmt.Sprintf("%v", err); !strings.Contains(got, expected) {
		t.Errorf("expected error containing %q, got %q", expected, got)
	}
}

func TestExecute_MissingModel(t *testing.T) {
	mock := newMockProvider("mock")
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	// No model on node or config
	f := &flow.Flow{
		Version: "1.0",
		Name:    "test-flow",
		Config: flow.Config{
			DefaultProvider: "mock",
			// DefaultModel intentionally empty
		},
		Nodes: []flow.Node{
			{
				ID:      "node1",
				Prompt:  "Hello",
				Inputs:  []flow.Input{{Name: "x", From: "input"}},
				Outputs: []flow.Output{{Name: "result", To: "output"}},
			},
		},
	}

	_, err := exec.Execute(context.Background(), f, map[string]any{"x": "val"})
	if err == nil {
		t.Fatal("expected error for missing model, got nil")
	}

	expected := "no model specified"
	if got := fmt.Sprintf("%v", err); !strings.Contains(got, expected) {
		t.Errorf("expected error containing %q, got %q", expected, got)
	}
}

// --- helpers ---

func nodeResultIDs(result *flow.ExecutionResult) []string {
	ids := make([]string, len(result.NodeResults))
	for i, nr := range result.NodeResults {
		ids[i] = nr.NodeID
	}
	return ids
}

func indexOf(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
}

func TestExecute_ProviderError(t *testing.T) {
	mock := newMockProvider("mock")
	mock.err = fmt.Errorf("rate limit exceeded")
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "node1",
			Prompt:  "Hello",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"x": "val"})
	if err == nil {
		t.Fatal("expected error from provider, got nil")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("expected error containing 'rate limit exceeded', got %q", err.Error())
	}
	if result != nil && result.Success {
		t.Error("expected Success=false")
	}
}

func TestExecute_NodeLevelProviderOverride(t *testing.T) {
	defaultMock := newMockProvider("default-provider")
	overrideMock := newMockProvider("override-provider")
	overrideMock.responses["Hello"] = "from override"

	registry := providers.NewRegistry()
	registry.Register(defaultMock)
	registry.Register(overrideMock)
	exec := New(registry)

	f := &flow.Flow{
		Version: "1.0",
		Name:    "test-flow",
		Config: flow.Config{
			DefaultProvider: "default-provider",
			DefaultModel:    "default-model",
		},
		Nodes: []flow.Node{
			{
				ID:       "node1",
				Provider: "override-provider",
				Model:    "override-model",
				Prompt:   "Hello",
				Inputs:   []flow.Input{{Name: "x", From: "input"}},
				Outputs:  []flow.Output{{Name: "result", To: "output"}},
			},
		},
	}

	result, err := exec.Execute(context.Background(), f, map[string]any{"x": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify the override provider was called, not the default
	if len(defaultMock.calls) != 0 {
		t.Errorf("expected 0 calls to default provider, got %d", len(defaultMock.calls))
	}
	if len(overrideMock.calls) != 1 {
		t.Fatalf("expected 1 call to override provider, got %d", len(overrideMock.calls))
	}
	if overrideMock.calls[0].Model != "override-model" {
		t.Errorf("expected model 'override-model', got %q", overrideMock.calls[0].Model)
	}
	if result.Outputs["result"] != "from override" {
		t.Errorf("expected output 'from override', got %q", result.Outputs["result"])
	}
}
