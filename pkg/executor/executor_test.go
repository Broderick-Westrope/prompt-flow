package executor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/broderick/prompt-flow/pkg/providers"
)

// mockProvider implements providers.Provider for testing.
type mockProvider struct {
	name      string
	responses map[string]string             // prompt content -> response content
	mu        sync.Mutex
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
	m.mu.Lock()
	m.calls = append(m.calls, req)
	m.mu.Unlock()

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

func findNodeResult(result *flow.ExecutionResult, nodeID string) *flow.NodeResult {
	for i := range result.NodeResults {
		if result.NodeResults[i].NodeID == nodeID {
			return &result.NodeResults[i]
		}
	}
	return nil
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

// --- Router node tests ---

func TestExecute_RouterSelectsCorrectBranch(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Classify: help me with billing"] = "billing"
	mock.responses["Handle billing: billing"] = "billing handled"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "classifier",
			Type:    "llm",
			Prompt:  "Classify: {{.user_input}}",
			Inputs:  []flow.Input{{Name: "user_input", From: "input"}},
			Outputs: []flow.Output{{Name: "category", To: ""}},
		},
		{
			ID:   "router",
			Type: "router",
			Inputs: []flow.Input{{Name: "category", From: "classifier.category"}},
			Routes: []flow.Route{
				{When: `== "billing"`, Next: "handle_billing"},
				{Default: true, Next: "handle_other"},
			},
		},
		{
			ID:      "handle_billing",
			Type:    "llm",
			Prompt:  "Handle billing: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "handle_other",
			Type:    "llm",
			Prompt:  "Handle other: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"user_input": "help me with billing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// handle_billing should have executed
	billingResult := findNodeResult(result, "handle_billing")
	if billingResult == nil {
		t.Fatal("expected handle_billing node result")
	}
	if billingResult.Skipped {
		t.Error("expected handle_billing to NOT be skipped")
	}

	// handle_other should be skipped
	otherResult := findNodeResult(result, "handle_other")
	if otherResult == nil {
		t.Fatal("expected handle_other node result")
	}
	if !otherResult.Skipped {
		t.Error("expected handle_other to be skipped")
	}
}

func TestExecute_RouterSkippedBranchNoLLMCall(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Classify: test"] = "billing"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "classifier",
			Type:    "llm",
			Prompt:  "Classify: {{.user_input}}",
			Inputs:  []flow.Input{{Name: "user_input", From: "input"}},
			Outputs: []flow.Output{{Name: "category", To: ""}},
		},
		{
			ID:   "router",
			Type: "router",
			Inputs: []flow.Input{{Name: "category", From: "classifier.category"}},
			Routes: []flow.Route{
				{When: `== "billing"`, Next: "handle_billing"},
				{Default: true, Next: "handle_other"},
			},
		},
		{
			ID:      "handle_billing",
			Type:    "llm",
			Prompt:  "Handle billing: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "handle_other",
			Type:    "llm",
			Prompt:  "Handle other: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	_, err := exec.Execute(context.Background(), f, map[string]any{"user_input": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mock should have been called exactly twice: classifier + handle_billing
	// handle_other should NOT have triggered a provider call
	if len(mock.calls) != 2 {
		t.Errorf("expected 2 provider calls (classifier + handle_billing), got %d", len(mock.calls))
	}
}

func TestExecute_RouterDefaultRouteUsed(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Classify: mystery"] = "unknown_category"
	mock.responses["Handle other: unknown_category"] = "other handled"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "classifier",
			Type:    "llm",
			Prompt:  "Classify: {{.user_input}}",
			Inputs:  []flow.Input{{Name: "user_input", From: "input"}},
			Outputs: []flow.Output{{Name: "category", To: ""}},
		},
		{
			ID:   "router",
			Type: "router",
			Inputs: []flow.Input{{Name: "category", From: "classifier.category"}},
			Routes: []flow.Route{
				{When: `== "billing"`, Next: "handle_billing"},
				{Default: true, Next: "handle_other"},
			},
		},
		{
			ID:      "handle_billing",
			Type:    "llm",
			Prompt:  "Handle billing: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "handle_other",
			Type:    "llm",
			Prompt:  "Handle other: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"user_input": "mystery"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// handle_other should have executed (default route)
	otherResult := findNodeResult(result, "handle_other")
	if otherResult == nil {
		t.Fatal("expected handle_other node result")
	}
	if otherResult.Skipped {
		t.Error("expected handle_other to NOT be skipped")
	}

	// handle_billing should be skipped
	billingResult := findNodeResult(result, "handle_billing")
	if billingResult == nil {
		t.Fatal("expected handle_billing node result")
	}
	if !billingResult.Skipped {
		t.Error("expected handle_billing to be skipped")
	}
}

func TestExecute_RouterNoMatchNoDefault(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Classify: mystery"] = "unknown"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "classifier",
			Type:    "llm",
			Prompt:  "Classify: {{.user_input}}",
			Inputs:  []flow.Input{{Name: "user_input", From: "input"}},
			Outputs: []flow.Output{{Name: "category", To: ""}},
		},
		{
			ID:   "router",
			Type: "router",
			Inputs: []flow.Input{{Name: "category", From: "classifier.category"}},
			Routes: []flow.Route{
				{When: `== "billing"`, Next: "handle_billing"},
				{When: `== "technical"`, Next: "handle_technical"},
			},
		},
		{
			ID:      "handle_billing",
			Type:    "llm",
			Prompt:  "Handle billing",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "handle_technical",
			Type:    "llm",
			Prompt:  "Handle technical",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	_, err := exec.Execute(context.Background(), f, map[string]any{"user_input": "mystery"})
	if err == nil {
		t.Fatal("expected error when no route matches and no default, got nil")
	}
	if !strings.Contains(err.Error(), "no route matched") {
		t.Errorf("expected error containing 'no route matched', got %q", err.Error())
	}
}

func TestExecute_RouterTransitiveSkipping(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Classify: test"] = "billing"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "classifier",
			Type:    "llm",
			Prompt:  "Classify: {{.user_input}}",
			Inputs:  []flow.Input{{Name: "user_input", From: "input"}},
			Outputs: []flow.Output{{Name: "category", To: ""}},
		},
		{
			ID:   "router",
			Type: "router",
			Inputs: []flow.Input{{Name: "category", From: "classifier.category"}},
			Routes: []flow.Route{
				{When: `== "billing"`, Next: "handle_billing"},
				{Default: true, Next: "handle_other"},
			},
		},
		{
			ID:      "handle_billing",
			Type:    "llm",
			Prompt:  "Handle billing: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "handle_other",
			Type:    "llm",
			Prompt:  "Handle other: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "follow_up",
			Type:    "llm",
			Prompt:  "Follow up: {{.other_result}}",
			Inputs:  []flow.Input{{Name: "other_result", From: "handle_other.result"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"user_input": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// handle_other should be skipped
	otherResult := findNodeResult(result, "handle_other")
	if otherResult == nil {
		t.Fatal("expected handle_other node result")
	}
	if !otherResult.Skipped {
		t.Error("expected handle_other to be skipped")
	}

	// follow_up depends on handle_other, so it should also be skipped
	followUpResult := findNodeResult(result, "follow_up")
	if followUpResult == nil {
		t.Fatal("expected follow_up node result")
	}
	if !followUpResult.Skipped {
		t.Error("expected follow_up to be skipped (transitive)")
	}

	// Only classifier + handle_billing should have made provider calls
	if len(mock.calls) != 2 {
		t.Errorf("expected 2 provider calls, got %d", len(mock.calls))
	}
}

// --- Parallel execution tests ---

func TestExecute_ParallelIndependentNodes(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Process A: hello"] = "result A"
	mock.responses["Process B: hello"] = "result B"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "nodeA",
			Prompt:  "Process A: {{.x}}",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "nodeB",
			Prompt:  "Process B: {{.x}}",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"x": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Both nodes should have executed
	if len(result.NodeResults) != 2 {
		t.Fatalf("expected 2 node results, got %d", len(result.NodeResults))
	}

	aResult := findNodeResult(result, "nodeA")
	bResult := findNodeResult(result, "nodeB")
	if aResult == nil || bResult == nil {
		t.Fatal("expected both nodeA and nodeB results")
	}
	if !aResult.Success || !bResult.Success {
		t.Error("expected both nodes to succeed")
	}

	mock.mu.Lock()
	callCount := len(mock.calls)
	mock.mu.Unlock()
	if callCount != 2 {
		t.Errorf("expected 2 provider calls, got %d", callCount)
	}
}

func TestExecute_ParallelDiamondPattern(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Start: val"] = "from A"
	mock.responses["B: from A"] = "from B"
	mock.responses["C: from A"] = "from C"
	mock.responses["D: from B from C"] = "from D"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "A",
			Prompt:  "Start: {{.x}}",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "B",
			Prompt:  "B: {{.a_out}}",
			Inputs:  []flow.Input{{Name: "a_out", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "C",
			Prompt:  "C: {{.a_out}}",
			Inputs:  []flow.Input{{Name: "a_out", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:     "D",
			Prompt: "D: {{.b_out}} {{.c_out}}",
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
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if len(result.NodeResults) != 4 {
		t.Fatalf("expected 4 node results, got %d", len(result.NodeResults))
	}

	dResult := findNodeResult(result, "D")
	if dResult == nil {
		t.Fatal("expected D node result")
	}
	if dResult.Outputs["result"] != "from D" {
		t.Errorf("expected D output 'from D', got %q", dResult.Outputs["result"])
	}

	if result.Outputs["result"] != "from D" {
		t.Errorf("expected flow output 'from D', got %q", result.Outputs["result"])
	}
}

func TestExecute_ParallelErrorCancelsLevel(t *testing.T) {
	mock := newMockProvider("mock")
	mock.err = fmt.Errorf("provider failure")
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "nodeA",
			Prompt:  "Process A: {{.x}}",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "nodeB",
			Prompt:  "Process B: {{.x}}",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"x": "val"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "provider failure") {
		t.Errorf("expected error containing 'provider failure', got %q", err.Error())
	}
	if result != nil && result.Success {
		t.Error("expected Success=false")
	}
}

func TestExecute_ParallelRouterSkipping(t *testing.T) {
	mock := newMockProvider("mock")
	mock.responses["Classify: test"] = "billing"
	mock.responses["Handle billing: billing"] = "billing handled"
	registry := providers.NewRegistry()
	registry.Register(mock)
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "classifier",
			Type:    "llm",
			Prompt:  "Classify: {{.user_input}}",
			Inputs:  []flow.Input{{Name: "user_input", From: "input"}},
			Outputs: []flow.Output{{Name: "category", To: ""}},
		},
		{
			ID:   "router",
			Type: "router",
			Inputs: []flow.Input{{Name: "category", From: "classifier.category"}},
			Routes: []flow.Route{
				{When: `== "billing"`, Next: "handle_billing"},
				{Default: true, Next: "handle_other"},
			},
		},
		{
			ID:      "handle_billing",
			Type:    "llm",
			Prompt:  "Handle billing: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "handle_other",
			Type:    "llm",
			Prompt:  "Handle other: {{.cat}}",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	result, err := exec.Execute(context.Background(), f, map[string]any{"user_input": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// handle_billing should have executed
	billingResult := findNodeResult(result, "handle_billing")
	if billingResult == nil {
		t.Fatal("expected handle_billing node result")
	}
	if billingResult.Skipped {
		t.Error("expected handle_billing to NOT be skipped")
	}

	// handle_other should be skipped
	otherResult := findNodeResult(result, "handle_other")
	if otherResult == nil {
		t.Fatal("expected handle_other node result")
	}
	if !otherResult.Skipped {
		t.Error("expected handle_other to be skipped")
	}

	// Only classifier + handle_billing should have made LLM calls
	mock.mu.Lock()
	callCount := len(mock.calls)
	mock.mu.Unlock()
	if callCount != 2 {
		t.Errorf("expected 2 provider calls (classifier + handle_billing), got %d", callCount)
	}
}

// --- executionLevels unit tests ---

func TestExecutionLevels_Basic(t *testing.T) {
	registry := providers.NewRegistry()
	exec := New(registry)

	// A → B, A → C, B → D, C → D
	f := baseFlow([]flow.Node{
		{
			ID:      "A",
			Prompt:  "A",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "B",
			Prompt:  "B",
			Inputs:  []flow.Input{{Name: "a", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:      "C",
			Prompt:  "C",
			Inputs:  []flow.Input{{Name: "a", From: "A.result"}},
			Outputs: []flow.Output{{Name: "result", To: ""}},
		},
		{
			ID:     "D",
			Prompt: "D",
			Inputs: []flow.Input{
				{Name: "b", From: "B.result"},
				{Name: "c", From: "C.result"},
			},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	levels, err := exec.executionLevels(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(levels))
	}

	// Level 0: [A]
	level0IDs := levelNodeIDs(levels[0])
	if len(level0IDs) != 1 || level0IDs[0] != "A" {
		t.Errorf("expected level 0 = [A], got %v", level0IDs)
	}

	// Level 1: [B, C] (order may vary)
	level1IDs := levelNodeIDs(levels[1])
	sort.Strings(level1IDs)
	if len(level1IDs) != 2 || level1IDs[0] != "B" || level1IDs[1] != "C" {
		t.Errorf("expected level 1 = [B, C], got %v", level1IDs)
	}

	// Level 2: [D]
	level2IDs := levelNodeIDs(levels[2])
	if len(level2IDs) != 1 || level2IDs[0] != "D" {
		t.Errorf("expected level 2 = [D], got %v", level2IDs)
	}
}

func TestExecutionLevels_WithRouter(t *testing.T) {
	registry := providers.NewRegistry()
	exec := New(registry)

	f := baseFlow([]flow.Node{
		{
			ID:      "classifier",
			Type:    "llm",
			Prompt:  "Classify",
			Inputs:  []flow.Input{{Name: "x", From: "input"}},
			Outputs: []flow.Output{{Name: "category", To: ""}},
		},
		{
			ID:   "router",
			Type: "router",
			Inputs: []flow.Input{{Name: "category", From: "classifier.category"}},
			Routes: []flow.Route{
				{When: `== "a"`, Next: "handle_a"},
				{Default: true, Next: "handle_b"},
			},
		},
		{
			ID:      "handle_a",
			Type:    "llm",
			Prompt:  "Handle A",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
		{
			ID:      "handle_b",
			Type:    "llm",
			Prompt:  "Handle B",
			Inputs:  []flow.Input{{Name: "cat", From: "classifier.category"}},
			Outputs: []flow.Output{{Name: "result", To: "output"}},
		},
	})

	levels, err := exec.executionLevels(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(levels))
	}

	// Level 0: [classifier]
	level0IDs := levelNodeIDs(levels[0])
	if len(level0IDs) != 1 || level0IDs[0] != "classifier" {
		t.Errorf("expected level 0 = [classifier], got %v", level0IDs)
	}

	// Level 1: [router]
	level1IDs := levelNodeIDs(levels[1])
	if len(level1IDs) != 1 || level1IDs[0] != "router" {
		t.Errorf("expected level 1 = [router], got %v", level1IDs)
	}

	// Level 2: [handle_a, handle_b] (order may vary)
	level2IDs := levelNodeIDs(levels[2])
	sort.Strings(level2IDs)
	if len(level2IDs) != 2 || level2IDs[0] != "handle_a" || level2IDs[1] != "handle_b" {
		t.Errorf("expected level 2 = [handle_a, handle_b], got %v", level2IDs)
	}
}

func levelNodeIDs(nodes []*flow.Node) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}
