package flow

import (
	"errors"
	"strings"
	"testing"
)

// validFlow returns a minimal valid flow for use as a test baseline.
func validFlow() *Flow {
	return &Flow{
		Version: "1.0",
		Name:    "test-flow",
		Nodes: []Node{
			{
				ID:      "node1",
				Prompt:  "Hello",
				Inputs:  []Input{{Name: "user_input", From: "input"}},
				Outputs: []Output{{Name: "result", To: "output"}},
			},
		},
	}
}

func TestValidate_ValidSingleNode(t *testing.T) {
	if err := Validate(validFlow()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_ValidMultiNodeWithDeps(t *testing.T) {
	f := &Flow{
		Version: "1.0",
		Name:    "multi-node",
		Nodes: []Node{
			{
				ID:      "a",
				Prompt:  "first prompt",
				Inputs:  []Input{{Name: "in", From: "input"}},
				Outputs: []Output{{Name: "out"}},
			},
			{
				ID:      "b",
				Prompt:  "second prompt",
				Inputs:  []Input{{Name: "in", From: "a.out"}},
				Outputs: []Output{{Name: "result", To: "output"}},
			},
		},
	}
	if err := Validate(f); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Flow)
		wantErr string
	}{
		{
			name:    "missing name",
			modify:  func(f *Flow) { f.Name = "" },
			wantErr: "name",
		},
		{
			name:    "missing version",
			modify:  func(f *Flow) { f.Version = "" },
			wantErr: "version",
		},
		{
			name:    "empty nodes",
			modify:  func(f *Flow) { f.Nodes = nil },
			wantErr: "nodes",
		},
		{
			name: "duplicate node IDs",
			modify: func(f *Flow) {
				f.Nodes = append(f.Nodes, Node{
					ID:      "node1",
					Prompt:  "duplicate",
					Inputs:  []Input{{Name: "in", From: "input"}},
					Outputs: []Output{{Name: "out"}},
				})
			},
			wantErr: "duplicate node ID",
		},
		{
			name: "missing node ID",
			modify: func(f *Flow) {
				f.Nodes[0].ID = ""
			},
			wantErr: "node ID is required",
		},
		{
			name: "missing prompt",
			modify: func(f *Flow) {
				f.Nodes[0].Prompt = ""
			},
			wantErr: "prompt is required",
		},
		{
			name: "missing output name",
			modify: func(f *Flow) {
				f.Nodes[0].Outputs = []Output{{Name: "", To: "output"}}
			},
			wantErr: "output name is required",
		},
		{
			name: "duplicate output names",
			modify: func(f *Flow) {
				f.Nodes[0].Outputs = []Output{
					{Name: "result", To: "output"},
					{Name: "result"},
				}
			},
			wantErr: "duplicate output name",
		},
		{
			name: "missing input name",
			modify: func(f *Flow) {
				f.Nodes[0].Inputs = []Input{{Name: "", From: "input"}}
			},
			wantErr: "input name is required",
		},
		{
			name: "missing input from",
			modify: func(f *Flow) {
				f.Nodes[0].Inputs = []Input{{Name: "in", From: ""}}
			},
			wantErr: "input source is required",
		},
		{
			name: "duplicate input names",
			modify: func(f *Flow) {
				f.Nodes[0].Inputs = []Input{
					{Name: "in", From: "input"},
					{Name: "in", From: "input"},
				}
			},
			wantErr: "duplicate input name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := validFlow()
			tt.modify(f)
			err := Validate(f)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); !strings.Contains(got, tt.wantErr) {
				t.Errorf("error = %q, want substring %q", got, tt.wantErr)
			}
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

func TestValidate_CycleDetection_Simple(t *testing.T) {
	f := &Flow{
		Version: "1.0",
		Name:    "cycle-test",
		Nodes: []Node{
			{
				ID:      "a",
				Prompt:  "prompt a",
				Inputs:  []Input{{Name: "in", From: "b.out"}},
				Outputs: []Output{{Name: "out"}},
			},
			{
				ID:      "b",
				Prompt:  "prompt b",
				Inputs:  []Input{{Name: "in", From: "a.out"}},
				Outputs: []Output{{Name: "out"}},
			},
		},
	}
	err := Validate(f)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error = %q, want substring 'cycle'", err.Error())
	}
}

func TestValidate_CycleDetection_LongChain(t *testing.T) {
	f := &Flow{
		Version: "1.0",
		Name:    "long-cycle",
		Nodes: []Node{
			{
				ID:      "a",
				Prompt:  "prompt",
				Inputs:  []Input{{Name: "in", From: "c.out"}},
				Outputs: []Output{{Name: "out"}},
			},
			{
				ID:      "b",
				Prompt:  "prompt",
				Inputs:  []Input{{Name: "in", From: "a.out"}},
				Outputs: []Output{{Name: "out"}},
			},
			{
				ID:      "c",
				Prompt:  "prompt",
				Inputs:  []Input{{Name: "in", From: "b.out"}},
				Outputs: []Output{{Name: "out"}},
			},
		},
	}
	err := Validate(f)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error = %q, want substring 'cycle'", err.Error())
	}
}

func TestValidate_Reference_MissingNode(t *testing.T) {
	f := &Flow{
		Version: "1.0",
		Name:    "ref-test",
		Nodes: []Node{
			{
				ID:      "a",
				Prompt:  "prompt",
				Inputs:  []Input{{Name: "in", From: "nonexistent.out"}},
				Outputs: []Output{{Name: "out"}},
			},
		},
	}
	err := Validate(f)
	if err == nil {
		t.Fatal("expected error for missing node reference, got nil")
	}
	if !strings.Contains(err.Error(), "referenced node does not exist") {
		t.Errorf("error = %q, want substring about missing node", err.Error())
	}
}

func TestValidate_Reference_MissingOutput(t *testing.T) {
	f := &Flow{
		Version: "1.0",
		Name:    "ref-test",
		Nodes: []Node{
			{
				ID:      "a",
				Prompt:  "prompt",
				Inputs:  []Input{{Name: "in", From: "input"}},
				Outputs: []Output{{Name: "out"}},
			},
			{
				ID:      "b",
				Prompt:  "prompt",
				Inputs:  []Input{{Name: "in", From: "a.nonexistent"}},
				Outputs: []Output{{Name: "out"}},
			},
		},
	}
	err := Validate(f)
	if err == nil {
		t.Fatal("expected error for missing output reference, got nil")
	}
	if !strings.Contains(err.Error(), "referenced output does not exist") {
		t.Errorf("error = %q, want substring about missing output", err.Error())
	}
}

func TestValidate_Reference_InvalidFormat(t *testing.T) {
	f := &Flow{
		Version: "1.0",
		Name:    "ref-test",
		Nodes: []Node{
			{
				ID:      "a",
				Prompt:  "prompt",
				Inputs:  []Input{{Name: "in", From: "no_dot_separator"}},
				Outputs: []Output{{Name: "out"}},
			},
		},
	}
	err := Validate(f)
	if err == nil {
		t.Fatal("expected error for invalid reference format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid input reference format") {
		t.Errorf("error = %q, want substring about invalid format", err.Error())
	}
}

func TestValidate_ReturnsValidationError(t *testing.T) {
	f := validFlow()
	f.Name = ""
	err := Validate(f)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

