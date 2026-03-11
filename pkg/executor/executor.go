package executor

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/broderick/prompt-flow/pkg/providers"
	"golang.org/x/sync/errgroup"
)

// Executor executes a flow
type Executor struct {
	registry       *providers.Registry
	MaxConcurrency int
}

func New(registry *providers.Registry) *Executor {
	return &Executor{
		registry:       registry,
		MaxConcurrency: 10,
	}
}

func (e *Executor) Execute(ctx context.Context, f *flow.Flow, inputs map[string]any) (*flow.ExecutionResult, error) {
	startTime := time.Now()

	result := &flow.ExecutionResult{
		FlowName:    f.Name,
		Success:     false,
		Outputs:     make(map[string]any),
		NodeResults: []flow.NodeResult{},
		StartTime:   startTime,
	}

	err := flow.Validate(f)
	if err != nil {
		result.Error = fmt.Sprintf("validation failed: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result, err
	}

	concurrency := e.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	levels, err := e.executionLevels(f)
	if err != nil {
		result.Error = fmt.Sprintf("failed to build execution order: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result, err
	}

	nodeOutputs := make(map[string]map[string]any)
	skippedNodes := make(map[string]bool)

	var mu sync.Mutex // protects result.NodeResults appends

	for _, level := range levels {
		// Resolve input data for all nodes on the main goroutine before
		// launching workers. This avoids concurrent reads of nodeOutputs
		// (which is only written to between levels, never during).
		type nodeWork struct {
			node      *flow.Node
			inputData map[string]any
		}
		var work []nodeWork

		for _, node := range level {
			if skippedNodes[node.ID] {
				nodeResult := &flow.NodeResult{
					NodeID:    node.ID,
					Success:   true,
					Skipped:   true,
					Outputs:   make(map[string]any),
					StartTime: time.Now(),
					EndTime:   time.Now(),
				}
				result.NodeResults = append(result.NodeResults, *nodeResult)
				nodeOutputs[node.ID] = nodeResult.Outputs
				continue
			}

			inputData, err := resolveInputData(node, inputs, nodeOutputs)
			if err != nil {
				result.Error = fmt.Sprintf("node %s failed: %v", node.ID, err)
				result.EndTime = time.Now()
				result.Duration = time.Since(startTime)
				return result, err
			}
			work = append(work, nodeWork{node: node, inputData: inputData})
		}

		if len(work) == 0 {
			continue
		}

		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(concurrency)

		for _, w := range work {
			g.Go(func() error {
				nodeResult, err := e.executeNode(gCtx, f, w.node, w.inputData)

				mu.Lock()
				result.NodeResults = append(result.NodeResults, *nodeResult)
				if err == nil {
					nodeOutputs[w.node.ID] = nodeResult.Outputs
				}
				mu.Unlock()

				if err != nil {
					return fmt.Errorf("node %s failed: %v", w.node.ID, err)
				}
				return nil
			})
		}

		// errgroup returns only the first error; other failures are
		// recorded in result.NodeResults for debugging.
		if err := g.Wait(); err != nil {
			result.Error = err.Error()
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime)
			return result, err
		}

		// After the level completes, process router skip decisions
		for _, node := range level {
			if node.Type == "router" && !skippedNodes[node.ID] {
				selectedNext, _ := nodeOutputs[node.ID]["selected"].(string)
				for _, route := range node.Routes {
					if route.Next != selectedNext {
						markSkipped(skippedNodes, route.Next, f)
					}
				}
			}
		}
	}

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

// resolveInputData builds the input data map for a node by reading from
// flow inputs and previous node outputs. Must be called from the main
// goroutine (not inside a worker) to avoid concurrent map reads.
func resolveInputData(node *flow.Node, flowInputs map[string]any, nodeOutputs map[string]map[string]any) (map[string]any, error) {
	inputData := make(map[string]any)
	for _, input := range node.Inputs {
		if input.From == "input" {
			if val, ok := flowInputs[input.Name]; ok {
				inputData[input.Name] = val
			} else {
				return nil, fmt.Errorf("flow input not provided: %s", input.Name)
			}
		} else {
			parts := strings.SplitN(input.From, ".", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid input reference: %s", input.From)
			}
			nodeID, outputName := parts[0], parts[1]
			if outputs, ok := nodeOutputs[nodeID]; ok {
				if val, ok := outputs[outputName]; ok {
					inputData[input.Name] = val
				} else {
					return nil, fmt.Errorf("output not found: %s.%s", nodeID, outputName)
				}
			} else {
				return nil, fmt.Errorf("node outputs not found: %s", nodeID)
			}
		}
	}
	return inputData, nil
}

func (e *Executor) executeNode(
	ctx context.Context,
	f *flow.Flow,
	node *flow.Node,
	inputData map[string]any,
) (*flow.NodeResult, error) {
	startTime := time.Now()

	result := &flow.NodeResult{
		NodeID:    node.ID,
		Success:   false,
		Outputs:   make(map[string]any),
		StartTime: startTime,
	}

	switch node.Type {
	case "", "llm":
		output, metrics, err := e.executeLLMNode(ctx, f, node, inputData)
		if err != nil {
			result.Error = err.Error()
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime)
			return result, err
		}
		result.Outputs = output
		result.Metrics = *metrics
	case "router":
		output, err := e.executeRouterNode(node, inputData)
		if err != nil {
			result.Error = err.Error()
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime)
			return result, err
		}
		result.Outputs = output
	default:
		err := fmt.Errorf("unknown node type: %s", node.Type)
		result.Error = err.Error()
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result, err
	}

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

func (e *Executor) executionLevels(f *flow.Flow) ([][]*flow.Node, error) {
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

	// Build graph from input references
	for _, node := range f.Nodes {
		for _, input := range node.Inputs {
			if input.From != "input" {
				parts := strings.SplitN(input.From, ".", 2)
				if len(parts) == 2 {
					sourceNode := parts[0]
					adjList[sourceNode] = append(adjList[sourceNode], node.ID)
					inDegree[node.ID]++
				}
			}
		}
	}

	// Add router route edges
	for _, node := range f.Nodes {
		if node.Type == "router" {
			for _, route := range node.Routes {
				if route.Next != "" {
					adjList[node.ID] = append(adjList[node.ID], route.Next)
					inDegree[route.Next]++
				}
			}
		}
	}

	// Kahn's algorithm with level grouping
	var levels [][]*flow.Node
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	processed := 0
	for len(queue) > 0 {
		// All nodes in the current queue form one level
		level := make([]*flow.Node, 0, len(queue))
		for _, nodeID := range queue {
			level = append(level, nodeMap[nodeID])
		}
		levels = append(levels, level)

		// Process this level's nodes and find next level
		nextQueue := []string{}
		for _, nodeID := range queue {
			processed++
			for _, neighbor := range adjList[nodeID] {
				inDegree[neighbor]--
				if inDegree[neighbor] == 0 {
					nextQueue = append(nextQueue, neighbor)
				}
			}
		}
		queue = nextQueue
	}

	if processed != len(f.Nodes) {
		return nil, fmt.Errorf("cycle detected in flow graph")
	}

	return levels, nil
}

func (e *Executor) executeRouterNode(node *flow.Node, inputData map[string]any) (map[string]any, error) {
	// Get the first input value as the routing value
	var routeValue string
	var found bool
	for _, input := range node.Inputs {
		if val, ok := inputData[input.Name]; ok {
			routeValue = strings.TrimSpace(fmt.Sprintf("%v", val))
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("router node %s: no input value available for routing", node.ID)
	}

	// Evaluate routes
	var defaultRoute *flow.Route
	for i := range node.Routes {
		route := &node.Routes[i]
		if route.Default {
			defaultRoute = route
			continue
		}

		matched, err := EvaluateRouteExpression(routeValue, route.When)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate route expression %q: %w", route.When, err)
		}
		if matched {
			return map[string]any{"selected": route.Next}, nil
		}
	}

	// Use default if available
	if defaultRoute != nil {
		return map[string]any{"selected": defaultRoute.Next}, nil
	}

	return nil, fmt.Errorf("no route matched for value %q and no default route defined", routeValue)
}

func markSkipped(skippedNodes map[string]bool, nodeID string, f *flow.Flow) {
	if skippedNodes[nodeID] {
		return
	}
	skippedNodes[nodeID] = true

	// Propagate through input dependencies
	for _, node := range f.Nodes {
		for _, input := range node.Inputs {
			if input.From != "input" {
				parts := strings.SplitN(input.From, ".", 2)
				if len(parts) == 2 && parts[0] == nodeID {
					markSkipped(skippedNodes, node.ID, f)
				}
			}
		}
	}

	// Propagate through router route edges (if skipped node is a router)
	for _, node := range f.Nodes {
		if node.ID == nodeID && node.Type == "router" {
			for _, route := range node.Routes {
				markSkipped(skippedNodes, route.Next, f)
			}
		}
	}
}
