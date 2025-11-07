package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/broderick/prompt-flow/pkg/executor"
	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/broderick/prompt-flow/pkg/providers"
)

type TestCmd struct {
	FlowFile string        `arg:"" help:"Path to flow definition file"`
	Input    []string      `short:"i" help:"Input values as key=value pairs"`
	Timeout  time.Duration `short:"t" default:"5m" help:"Execution timeout"`
}

func (c *TestCmd) Run() error {
	// Parse the flow
	f, err := flow.Parse(c.FlowFile)
	if err != nil {
		return fmt.Errorf("failed to parse flow: %w", err)
	}

	// Validate the flow
	if err := flow.Validate(f); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Parse inputs
	inputs := make(map[string]any)
	for _, arg := range c.Input {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid input format: %s (expected key=value)", arg)
		}
		_, ok := inputs[parts[0]]
		if ok {
			fmt.Printf("warning: input %s already defined, overriding with value %s\n", parts[0], parts[1])
		}
		inputs[parts[0]] = parts[1]
	}

	// Create provider registry
	registry := providers.NewRegistry()
	registry.Register(providers.NewOpenAIProvider(os.Getenv("OPENAI_API_KEY")))
	registry.Register(providers.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY")))
	registry.Register(providers.NewGithubPlaygroundOpenAIProvider(os.Getenv("GITHUB_PLAYGROUND_PAT")))

	// Create executor
	exec := executor.New(registry)

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	fmt.Printf("Executing flow '%s'...\n\n", f.Name)

	result, err := exec.Execute(ctx, f, inputs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution failed: %v\n\n", err)
	}

	printExecutionResult(result)
	return err
}

func printExecutionResult(result *flow.ExecutionResult) {
	fmt.Printf("=== Execution Result ===\n")
	fmt.Printf("Flow: %s\n", result.FlowName)
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Duration: %v\n", result.Duration)

	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
	}

	fmt.Printf("\n=== Node Results ===\n")
	totalTokens := 0
	totalCost := 0.0

	for i, nodeResult := range result.NodeResults {
		fmt.Printf("\n[%d] Node: %s\n", i+1, nodeResult.NodeID)
		fmt.Printf("    Success: %v\n", nodeResult.Success)
		fmt.Printf("    Duration: %v\n", nodeResult.Duration)

		if nodeResult.Error != "" {
			fmt.Printf("    Error: %s\n", nodeResult.Error)
		}

		if nodeResult.Metrics.InputTokens > 0 {
			fmt.Printf("    Tokens: %d (input: %d, output: %d)\n",
				nodeResult.Metrics.InputTokens+nodeResult.Metrics.OutputTokens,
				nodeResult.Metrics.InputTokens,
				nodeResult.Metrics.OutputTokens)

			totalTokens += nodeResult.Metrics.InputTokens
		}

		if nodeResult.Metrics.InputCost > 0 {
			fmt.Printf("    Cost: $%.6f (input: $%.6f, output: $%.6f)\n",
				nodeResult.Metrics.InputCost+nodeResult.Metrics.OutputCost,
				nodeResult.Metrics.InputCost,
				nodeResult.Metrics.OutputCost)

			totalCost += nodeResult.Metrics.InputCost + nodeResult.Metrics.OutputCost
		}

		if len(nodeResult.Outputs) > 0 {
			fmt.Printf("    Outputs:\n")
			for key, val := range nodeResult.Outputs {
				fmt.Printf("      %s: %v\n", key, val)
			}
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total Tokens: %d\n", totalTokens)

	if totalCost > 0 {
		fmt.Printf("Total Cost: $%.6f\n", totalCost)
	}

	if len(result.Outputs) > 0 {
		fmt.Printf("\n=== Flow Outputs ===\n")
		outputJSON, _ := json.MarshalIndent(result.Outputs, "", "  ")
		fmt.Println(string(outputJSON))
	}
}
