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
		if err := validateNode(&node, nodeIDs); err != nil {
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

func validateNode(node *Node, existingIDs map[string]bool) error {
	if node.Type == "" {
		return ValidationError{Field: "type", Message: "node type is required"}
	}

	if node.Type == "llm" && node.Prompt == "" {
		return ValidationError{Field: "prompt", Message: "prompt is required for LLM nodes"}
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

func validateReferences(flow *Flow) error {
	// Build a map of available outputs
	availableOutputs := make(map[string]map[string]bool) // nodeID -> outputName -> true

	for _, node := range flow.Nodes {
		availableOutputs[node.ID] = make(map[string]bool)
		for _, output := range node.Outputs {
			availableOutputs[node.ID][output.Name] = true
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
		for _, input := range node.Inputs {
			if input.From != "input" {
				parts := strings.SplitN(input.From, ".", 2)
				if len(parts) == 2 {
					graph[node.ID] = append(graph[node.ID], parts[0])
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
