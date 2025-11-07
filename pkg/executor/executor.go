package executor

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/broderick/prompt-flow/pkg/providers"
)

// Executor executes a flow
type Executor struct {
	registry *providers.Registry
}

// New creates a new executor
func New(registry *providers.Registry) *Executor {
	return &Executor{
		registry: registry,
	}
}

// Execute runs a flow with the given inputs
func (e *Executor) Execute(ctx context.Context, f *flow.Flow, inputs map[string]any) (*flow.ExecutionResult, error) {
	startTime := time.Now()

	result := &flow.ExecutionResult{
		FlowName:    f.Name,
		Success:     false,
		Outputs:     make(map[string]any),
		NodeResults: []flow.NodeResult{},
		StartTime:   startTime,
	}

	// Validate flow first
	if err := flow.Validate(f); err != nil {
		result.Error = fmt.Sprintf("validation failed: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Build execution order using topological sort
	execOrder, err := e.topologicalSort(f)
	if err != nil {
		result.Error = fmt.Sprintf("failed to build execution order: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Storage for node outputs
	nodeOutputs := make(map[string]map[string]any) // nodeID -> outputName -> value

	// Execute nodes in order
	for _, node := range execOrder {
		nodeResult, err := e.executeNode(ctx, f, node, inputs, nodeOutputs)
		result.NodeResults = append(result.NodeResults, *nodeResult)

		if err != nil {
			result.Error = fmt.Sprintf("node %s failed: %v", node.ID, err)
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime)
			return result, err
		}

		// Store node outputs
		nodeOutputs[node.ID] = nodeResult.Outputs
	}

	// Collect flow outputs
	for _, node := range f.Nodes {
		for _, output := range node.Outputs {
			if output.To == "output" {
				result.Outputs[output.Name] = nodeOutputs[node.ID][output.Name]
			}
		}
	}

	result.Success = true
	result.EndTime = time.Now()
	result.Duration = time.Since(startTime)

	return result, nil
}

func (e *Executor) executeNode(
	ctx context.Context,
	f *flow.Flow,
	node *flow.Node,
	flowInputs map[string]any,
	nodeOutputs map[string]map[string]any,
) (*flow.NodeResult, error) {
	startTime := time.Now()

	result := &flow.NodeResult{
		NodeID:    node.ID,
		Success:   false,
		Outputs:   make(map[string]any),
		StartTime: startTime,
	}

	// Build input data for this node
	inputData := make(map[string]any)
	for _, input := range node.Inputs {
		if input.From == "input" {
			// Get from flow inputs
			if val, ok := flowInputs[input.Name]; ok {
				inputData[input.Name] = val
			} else {
				err := fmt.Errorf("flow input not provided: %s", input.Name)
				result.Error = err.Error()
				result.EndTime = time.Now()
				result.Duration = time.Since(startTime)
				return result, err
			}
		} else {
			// Get from another node's output
			parts := strings.SplitN(input.From, ".", 2)
			if len(parts) != 2 {
				err := fmt.Errorf("invalid input reference: %s", input.From)
				result.Error = err.Error()
				result.EndTime = time.Now()
				result.Duration = time.Since(startTime)
				return result, err
			}

			nodeID, outputName := parts[0], parts[1]
			if outputs, ok := nodeOutputs[nodeID]; ok {
				if val, ok := outputs[outputName]; ok {
					inputData[input.Name] = val
				} else {
					err := fmt.Errorf("output not found: %s.%s", nodeID, outputName)
					result.Error = err.Error()
					result.EndTime = time.Now()
					result.Duration = time.Since(startTime)
					return result, err
				}
			} else {
				err := fmt.Errorf("node outputs not found: %s", nodeID)
				result.Error = err.Error()
				result.EndTime = time.Now()
				result.Duration = time.Since(startTime)
				return result, err
			}
		}
	}

	// TODO: support other node types?
	output, metrics, err := e.executeLLMNode(ctx, f, node, inputData)
	if err != nil {
		result.Error = err.Error()
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result, err
	}
	result.Outputs = output
	result.Metrics = *metrics

	result.Success = true
	result.EndTime = time.Now()
	result.Duration = time.Since(startTime)

	return result, nil
}

func (e *Executor) executeLLMNode(
	ctx context.Context,
	f *flow.Flow,
	node *flow.Node,
	inputData map[string]any,
) (map[string]any, *flow.NodeMetrics, error) {
	// Render prompt template
	tmpl, err := template.New(node.ID).Parse(node.Prompt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var promptBuf bytes.Buffer
	if err := tmpl.Execute(&promptBuf, inputData); err != nil {
		return nil, nil, fmt.Errorf("failed to execute prompt template: %w", err)
	}

	prompt := promptBuf.String()

	// Get provider
	providerName := node.Provider
	if providerName == "" {
		providerName = f.Config.DefaultProvider
	}
	if providerName == "" {
		return nil, nil, fmt.Errorf("no provider specified for node and no default provider set")
	}

	provider, ok := e.registry.Get(providerName)
	if !ok {
		return nil, nil, fmt.Errorf("provider not found: %s", providerName)
	}

	// Get model
	model := node.Model
	if model == "" {
		model = f.Config.DefaultModel
	}
	if model == "" {
		return nil, nil, fmt.Errorf("no model specified for node and no default model set")
	}

	// Call LLM
	req := providers.CompletionRequest{
		Prompt:   prompt,
		Model:    model,
		Settings: node.Settings,
	}

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Build outputs
	outputs := make(map[string]any)
	if len(node.Outputs) > 0 {
		// For now, assume first output gets the response content
		// In the future, we could parse structured outputs
		outputs[node.Outputs[0].Name] = resp.Content
	}

	metrics := &flow.NodeMetrics{
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		InputCost:    resp.InputCost,
		OutputCost:   resp.OutputCost,
	}

	return outputs, metrics, nil
}

func (e *Executor) topologicalSort(f *flow.Flow) ([]*flow.Node, error) {
	// Build adjacency list and in-degree map
	adjList := make(map[string][]string)
	inDegree := make(map[string]int)
	nodeMap := make(map[string]*flow.Node)

	// Initialize
	for i := range f.Nodes {
		node := &f.Nodes[i]
		nodeMap[node.ID] = node
		adjList[node.ID] = []string{}
		inDegree[node.ID] = 0
	}

	// Build graph
	for _, node := range f.Nodes {
		for _, input := range node.Inputs {
			if input.From != "input" {
				parts := strings.SplitN(input.From, ".", 2)
				if len(parts) == 2 {
					sourceNode := parts[0]
					// Add edge from sourceNode to current node
					adjList[sourceNode] = append(adjList[sourceNode], node.ID)
					inDegree[node.ID]++
				}
			}
		}
	}

	// Kahn's algorithm for topological sort
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	result := []*flow.Node{}
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]

		result = append(result, nodeMap[nodeID])

		for _, neighbor := range adjList[nodeID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(f.Nodes) {
		return nil, fmt.Errorf("cycle detected in flow graph")
	}

	return result, nil
}
