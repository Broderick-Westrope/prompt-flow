package main

import (
	"context"
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
