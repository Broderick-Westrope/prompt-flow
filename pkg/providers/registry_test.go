package providers

import (
	"context"
	"sort"
	"testing"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Complete(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{Content: "mock"}, nil
}

func TestNewRegistry_Empty(t *testing.T) {
	r := NewRegistry()

	names := r.List()
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}

	p, ok := r.Get("anything")
	if ok {
		t.Error("expected ok to be false for empty registry")
	}
	if p != nil {
		t.Error("expected nil provider for empty registry")
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	mock := &mockProvider{name: "test-provider"}

	r.Register(mock)

	got, ok := r.Get("test-provider")
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if got.Name() != "test-provider" {
		t.Errorf("expected name %q, got %q", "test-provider", got.Name())
	}
}

func TestRegistry_GetUnregistered(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "exists"})

	tests := []struct {
		name    string
		lookupName string
	}{
		{"completely unknown", "unknown"},
		{"empty string", ""},
		{"similar name", "exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := r.Get(tt.lookupName)
			if ok {
				t.Errorf("expected ok=false for %q", tt.lookupName)
			}
			if p != nil {
				t.Errorf("expected nil provider for %q", tt.lookupName)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	names := []string{"alpha", "beta", "gamma"}
	for _, n := range names {
		r.Register(&mockProvider{name: n})
	}

	got := r.List()
	sort.Strings(got)
	sort.Strings(names)

	if len(got) != len(names) {
		t.Fatalf("expected %d names, got %d", len(names), len(got))
	}
	for i := range names {
		if got[i] != names[i] {
			t.Errorf("expected %q at index %d, got %q", names[i], i, got[i])
		}
	}
}

func TestRegistry_RegisterOverwrites(t *testing.T) {
	r := NewRegistry()

	first := &mockProvider{name: "same"}
	second := &mockProvider{name: "same"}

	r.Register(first)
	r.Register(second)

	got, ok := r.Get("same")
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if got != second {
		t.Error("expected Get to return the second registered provider")
	}

	names := r.List()
	if len(names) != 1 {
		t.Errorf("expected 1 provider after overwrite, got %d", len(names))
	}
}

func TestRegistry_MultipleIndependent(t *testing.T) {
	r := NewRegistry()
	providers := []*mockProvider{
		{name: "openai"},
		{name: "anthropic"},
		{name: "github"},
	}
	for _, p := range providers {
		r.Register(p)
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			got, ok := r.Get(p.name)
			if !ok {
				t.Fatalf("expected ok=true for %q", p.name)
			}
			if got != p {
				t.Errorf("expected exact provider instance for %q", p.name)
			}
		})
	}
}
