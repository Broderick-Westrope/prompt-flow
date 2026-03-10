package flow

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks if a flow definition is valid
func Validate(flow *Flow) error {
	if flow.Name == "" {
		return ValidationError{Field: "name", Message: "flow name is required"}
	}

	if flow.Version == "" {
		return ValidationError{Field: "version", Message: "flow version is required"}
	}

	if len(flow.Nodes) == 0 {
		return ValidationError{Field: "nodes", Message: "at least one node is required"}
	}

	// Build complete set of all node IDs (for forward reference validation)
	allNodeIDs := make(map[string]bool)
	for _, node := range flow.Nodes {
		if node.ID != "" {
			allNodeIDs[node.ID] = true
		}
	}

	// Check for unique node IDs
	nodeIDs := make(map[string]bool)
	for i, node := range flow.Nodes {
		if node.ID == "" {
			return ValidationError{
				Field:   fmt.Sprintf("nodes[%d].id", i),
				Message: "node ID is required",
			}
		}

		if nodeIDs[node.ID] {
			return ValidationError{
				Field:   fmt.Sprintf("nodes[%d].id", i),
				Message: fmt.Sprintf("duplicate node ID: %s", node.ID),
			}
		}
		nodeIDs[node.ID] = true

		// Validate node
		if err := validateNode(&node, nodeIDs, allNodeIDs); err != nil {
			return fmt.Errorf("node %s: %w", node.ID, err)
		}
	}

	// Check for cycles in the DAG
	if err := checkCycles(flow); err != nil {
		return err
	}

	// Validate all input references exist
	if err := validateReferences(flow); err != nil {
		return err
	}

	return nil
}

func validateNode(node *Node, existingIDs map[string]bool, allNodeIDs map[string]bool) error {
	switch node.Type {
	case "", "llm":
		return validateLLMNode(node, existingIDs)
	case "router":
		return validateRouterNode(node, allNodeIDs)
	default:
		return ValidationError{Field: "type", Message: fmt.Sprintf("unknown node type: %s", node.Type)}
	}
}

func validateLLMNode(node *Node, existingIDs map[string]bool) error {
	if node.Prompt == "" {
		return ValidationError{Field: "prompt", Message: "prompt is required"}
	}

	// Validate outputs
	outputNames := make(map[string]bool)
	for i, output := range node.Outputs {
		if output.Name == "" {
			return ValidationError{
				Field:   fmt.Sprintf("outputs[%d].name", i),
				Message: "output name is required",
			}
		}
		if outputNames[output.Name] {
			return ValidationError{
				Field:   fmt.Sprintf("outputs[%d].name", i),
				Message: fmt.Sprintf("duplicate output name: %s", output.Name),
			}
		}
		outputNames[output.Name] = true
	}

	// Validate inputs
	inputNames := make(map[string]bool)
	for i, input := range node.Inputs {
		if input.Name == "" {
			return ValidationError{
				Field:   fmt.Sprintf("inputs[%d].name", i),
				Message: "input name is required",
			}
		}
		if input.From == "" {
			return ValidationError{
				Field:   fmt.Sprintf("inputs[%d].from", i),
				Message: "input source is required",
			}
		}
		if inputNames[input.Name] {
			return ValidationError{
				Field:   fmt.Sprintf("inputs[%d].name", i),
				Message: fmt.Sprintf("duplicate input name: %s", input.Name),
			}
		}
		inputNames[input.Name] = true
	}

	return nil
}

func validateRouterNode(node *Node, allNodeIDs map[string]bool) error {
	// Router nodes must not have a prompt
	if node.Prompt != "" {
		return ValidationError{Field: "prompt", Message: "router nodes must not have a prompt"}
	}

	// Must have at least one input
	if len(node.Inputs) == 0 {
		return ValidationError{Field: "inputs", Message: "router nodes must have at least one input"}
	}

	// Validate inputs
	for i, input := range node.Inputs {
		if input.Name == "" {
			return ValidationError{
				Field:   fmt.Sprintf("inputs[%d].name", i),
				Message: "input name is required",
			}
		}
		if input.From == "" {
			return ValidationError{
				Field:   fmt.Sprintf("inputs[%d].from", i),
				Message: "input source is required",
			}
		}
	}

	// Must have at least one route
	if len(node.Routes) == 0 {
		return ValidationError{Field: "routes", Message: "router nodes must have at least one route"}
	}

	defaultCount := 0
	for i, route := range node.Routes {
		// Each route must have a next target
		if route.Next == "" {
			return ValidationError{
				Field:   fmt.Sprintf("routes[%d].next", i),
				Message: "route must have a next target",
			}
		}

		// Non-default routes must have a when expression
		if !route.Default && route.When == "" {
			return ValidationError{
				Field:   fmt.Sprintf("routes[%d].when", i),
				Message: "non-default route must have a when expression",
			}
		}

		// Count defaults
		if route.Default {
			defaultCount++
		}

		// Validate next target references an existing node
		if !allNodeIDs[route.Next] {
			return ValidationError{
				Field:   fmt.Sprintf("routes[%d].next", i),
				Message: fmt.Sprintf("route next target does not exist: %s", route.Next),
			}
		}
	}

	// At most one default route
	if defaultCount > 1 {
		return ValidationError{Field: "routes", Message: "router nodes must have at most one default route"}
	}

	return nil
}

func validateReferences(flow *Flow) error {
	// Build a map of available outputs
	availableOutputs := make(map[string]map[string]bool) // nodeID -> outputName -> true

	for _, node := range flow.Nodes {
		availableOutputs[node.ID] = make(map[string]bool)
		for _, output := range node.Outputs {
			availableOutputs[node.ID][output.Name] = true
		}
		// Router nodes expose a synthetic "selected" output
		if node.Type == "router" {
			availableOutputs[node.ID]["selected"] = true
		}
	}

	// Check all input references
	for _, node := range flow.Nodes {
		for _, input := range node.Inputs {
			if input.From == "input" {
				// This is a flow input, always valid
				continue
			}

			// Parse the reference (format: "nodeID.outputName")
			parts := strings.SplitN(input.From, ".", 2)
			if len(parts) != 2 {
				return ValidationError{
					Field:   fmt.Sprintf("node %s, input %s", node.ID, input.Name),
					Message: fmt.Sprintf("invalid input reference format: %s (expected 'nodeID.outputName')", input.From),
				}
			}

			nodeID, outputName := parts[0], parts[1]

			// Check if the referenced node exists
			if _, exists := availableOutputs[nodeID]; !exists {
				return ValidationError{
					Field:   fmt.Sprintf("node %s, input %s", node.ID, input.Name),
					Message: fmt.Sprintf("referenced node does not exist: %s", nodeID),
				}
			}

			// Check if the referenced output exists
			if !availableOutputs[nodeID][outputName] {
				return ValidationError{
					Field:   fmt.Sprintf("node %s, input %s", node.ID, input.Name),
					Message: fmt.Sprintf("referenced output does not exist: %s.%s", nodeID, outputName),
				}
			}
		}
	}

	return nil
}

func checkCycles(flow *Flow) error {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, node := range flow.Nodes {
		graph[node.ID] = []string{}
		// Input dependencies: edge from input source -> this node
		for _, input := range node.Inputs {
			if input.From != "input" {
				parts := strings.SplitN(input.From, ".", 2)
				if len(parts) == 2 {
					graph[node.ID] = append(graph[node.ID], parts[0])
				}
			}
		}
		// Router route edges: route targets depend on the router
		if node.Type == "router" {
			for _, route := range node.Routes {
				if route.Next != "" {
					graph[route.Next] = append(graph[route.Next], node.ID)
				}
			}
		}
	}

	// DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		for _, neighbor := range graph[nodeID] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range graph {
		if !visited[nodeID] {
			if hasCycle(nodeID) {
				return ValidationError{
					Field:   "nodes",
					Message: fmt.Sprintf("cycle detected in flow graph involving node: %s", nodeID),
				}
			}
		}
	}

	return nil
}
