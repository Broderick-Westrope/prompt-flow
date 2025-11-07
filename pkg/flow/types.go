package flow

import "time"

// Flow represents a complete prompt flow definition
type Flow struct {
	Version     string `yaml:"version" json:"version"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Config      Config `yaml:"config,omitempty" json:"config,omitempty"`
	Nodes       []Node `yaml:"nodes" json:"nodes"`
}

// Config holds flow-level configuration
type Config struct {
	DefaultProvider string            `yaml:"default_provider,omitempty" json:"default_provider,omitempty"`
	DefaultModel    string            `yaml:"default_model,omitempty" json:"default_model,omitempty"`
	Settings        map[string]string `yaml:"settings,omitempty" json:"settings,omitempty"`
}

// Node represents a single node in the flow
type Node struct {
	ID       string         `yaml:"id" json:"id"`
	Provider string         `yaml:"provider,omitempty" json:"provider,omitempty"`
	Model    string         `yaml:"model,omitempty" json:"model,omitempty"`
	Inputs   []Input        `yaml:"inputs" json:"inputs"`
	Prompt   string         `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	Outputs  []Output       `yaml:"outputs" json:"outputs"`
	Settings map[string]any `yaml:"settings,omitempty" json:"settings,omitempty"`
}

// Input represents an input to a node
type Input struct {
	Name string `yaml:"name" json:"name"`
	From string `yaml:"from" json:"from"` // "input" for flow input, or "node_id.output_name"
}

// Output represents an output from a node
type Output struct {
	Name string `yaml:"name" json:"name"`
	To   string `yaml:"to,omitempty" json:"to,omitempty"` // "output" for flow output, or empty
}

// ExecutionResult represents the result of executing a flow
type ExecutionResult struct {
	FlowName    string         `json:"flow_name"`
	Success     bool           `json:"success"`
	Error       string         `json:"error,omitempty"`
	Outputs     map[string]any `json:"outputs"`
	NodeResults []NodeResult   `json:"node_results"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	Duration    time.Duration  `json:"duration"`
}

// NodeResult represents the result of executing a single node
type NodeResult struct {
	NodeID    string         `json:"node_id"`
	Success   bool           `json:"success"`
	Error     string         `json:"error,omitempty"`
	Outputs   map[string]any `json:"outputs"`
	Metrics   NodeMetrics    `json:"metrics"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time"`
	Duration  time.Duration  `json:"duration"`
}

// NodeMetrics contains performance metrics for a node execution
type NodeMetrics struct {
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	InputCost    float64 `json:"input_cost,omitempty"`
	OutputCost   float64 `json:"output_cost,omitempty"`
}
