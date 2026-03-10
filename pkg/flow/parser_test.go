package flow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBytes_YAML(t *testing.T) {
	yamlData := []byte(`
version: "1.0"
name: test-flow
description: A test flow
config:
  default_provider: openai
  default_model: gpt-4
nodes:
  - id: node1
    prompt: "Hello {{.user_input}}"
    inputs:
      - name: user_input
        from: input
    outputs:
      - name: result
        to: output
`)
	flow, err := ParseBytes(yamlData, "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.Version != "1.0" {
		t.Errorf("version = %q, want %q", flow.Version, "1.0")
	}
	if flow.Name != "test-flow" {
		t.Errorf("name = %q, want %q", flow.Name, "test-flow")
	}
	if flow.Description != "A test flow" {
		t.Errorf("description = %q, want %q", flow.Description, "A test flow")
	}
	if flow.Config.DefaultProvider != "openai" {
		t.Errorf("default_provider = %q, want %q", flow.Config.DefaultProvider, "openai")
	}
	if flow.Config.DefaultModel != "gpt-4" {
		t.Errorf("default_model = %q, want %q", flow.Config.DefaultModel, "gpt-4")
	}
	if len(flow.Nodes) != 1 {
		t.Fatalf("len(nodes) = %d, want 1", len(flow.Nodes))
	}
	node := flow.Nodes[0]
	if node.ID != "node1" {
		t.Errorf("node.ID = %q, want %q", node.ID, "node1")
	}
	if node.Prompt != "Hello {{.user_input}}" {
		t.Errorf("node.Prompt = %q, want %q", node.Prompt, "Hello {{.user_input}}")
	}
	if len(node.Inputs) != 1 || node.Inputs[0].Name != "user_input" || node.Inputs[0].From != "input" {
		t.Errorf("unexpected inputs: %+v", node.Inputs)
	}
	if len(node.Outputs) != 1 || node.Outputs[0].Name != "result" || node.Outputs[0].To != "output" {
		t.Errorf("unexpected outputs: %+v", node.Outputs)
	}
}

func TestParseBytes_JSON(t *testing.T) {
	jsonData := []byte(`{
  "version": "1.0",
  "name": "json-flow",
  "nodes": [
    {
      "id": "node1",
      "prompt": "Hello",
      "inputs": [{"name": "user_input", "from": "input"}],
      "outputs": [{"name": "result", "to": "output"}]
    }
  ]
}`)
	flow, err := ParseBytes(jsonData, "test.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.Name != "json-flow" {
		t.Errorf("name = %q, want %q", flow.Name, "json-flow")
	}
	if len(flow.Nodes) != 1 {
		t.Fatalf("len(nodes) = %d, want 1", len(flow.Nodes))
	}
	if flow.Nodes[0].ID != "node1" {
		t.Errorf("node ID = %q, want %q", flow.Nodes[0].ID, "node1")
	}
}

func TestParseBytes_UnknownExtension_ValidYAML(t *testing.T) {
	yamlData := []byte(`
version: "1.0"
name: fallback-flow
nodes:
  - id: n1
    prompt: "test"
    inputs: []
    outputs: []
`)
	flow, err := ParseBytes(yamlData, "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.Name != "fallback-flow" {
		t.Errorf("name = %q, want %q", flow.Name, "fallback-flow")
	}
}

func TestParseBytes_MalformedYAML(t *testing.T) {
	data := []byte(`
version: "1.0"
name: [invalid yaml
  - this is broken
`)
	_, err := ParseBytes(data, "bad.yaml")
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestParseBytes_MalformedJSON(t *testing.T) {
	data := []byte(`{"version": "1.0", "name": broken json}`)
	_, err := ParseBytes(data, "bad.json")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestParse_YAML_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flow.yaml")
	content := []byte(`
version: "1.0"
name: file-flow
nodes:
  - id: n1
    prompt: "hello"
    inputs: []
    outputs: []
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	flow, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.Name != "file-flow" {
		t.Errorf("name = %q, want %q", flow.Name, "file-flow")
	}
}

func TestParse_JSON_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flow.json")
	content := []byte(`{
  "version": "1.0",
  "name": "json-file-flow",
  "nodes": [{"id": "n1", "prompt": "hello", "inputs": [], "outputs": []}]
}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	flow, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.Name != "json-file-flow" {
		t.Errorf("name = %q, want %q", flow.Name, "json-file-flow")
	}
}

func TestParse_NonexistentFile(t *testing.T) {
	_, err := Parse("/nonexistent/path/flow.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestSave_And_Reparse(t *testing.T) {
	original := &Flow{
		Version:     "1.0",
		Name:        "save-test",
		Description: "testing save",
		Config: Config{
			DefaultProvider: "openai",
			DefaultModel:    "gpt-4",
		},
		Nodes: []Node{
			{
				ID:     "n1",
				Prompt: "Hello {{.name}}",
				Inputs: []Input{{Name: "name", From: "input"}},
				Outputs: []Output{{Name: "greeting", To: "output"}},
			},
		},
	}

	tests := []struct {
		name     string
		filename string
	}{
		{"YAML", "flow.yaml"},
		{"JSON", "flow.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tt.filename)

			if err := Save(original, path); err != nil {
				t.Fatalf("Save() error: %v", err)
			}

			parsed, err := Parse(path)
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			if parsed.Name != original.Name {
				t.Errorf("name = %q, want %q", parsed.Name, original.Name)
			}
			if parsed.Version != original.Version {
				t.Errorf("version = %q, want %q", parsed.Version, original.Version)
			}
			if parsed.Description != original.Description {
				t.Errorf("description = %q, want %q", parsed.Description, original.Description)
			}
			if parsed.Config.DefaultProvider != original.Config.DefaultProvider {
				t.Errorf("default_provider = %q, want %q", parsed.Config.DefaultProvider, original.Config.DefaultProvider)
			}
			if len(parsed.Nodes) != 1 {
				t.Fatalf("len(nodes) = %d, want 1", len(parsed.Nodes))
			}
			if parsed.Nodes[0].ID != "n1" {
				t.Errorf("node ID = %q, want %q", parsed.Nodes[0].ID, "n1")
			}
		})
	}
}

func TestSave_UnsupportedExtension(t *testing.T) {
	flow := &Flow{Name: "test", Version: "1.0"}
	dir := t.TempDir()
	path := filepath.Join(dir, "flow.txt")

	err := Save(flow, path)
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
}
