package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"strconv"

	"github.com/broderick/prompt-flow/pkg/executor"
	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/broderick/prompt-flow/pkg/providers"
)

//go:embed simple.flow.yaml
var simpleFlow []byte

func main() {
	err := run()
	if err != nil {
		log.Fatalf("failed to run: %v", err)
	}
}

func run() (err error) {
	f, err := flow.ParseBytes(simpleFlow, "simple.flow.yaml")
	if err != nil {
		return fmt.Errorf("parsing flow: %w", err)
	}

	err = flow.Validate(f)
	if err != nil {
		return fmt.Errorf("validating flow: %w", err)
	}

	registry := providers.NewRegistry().WithDefaultProviders()
	exec := executor.New(registry)

	inputs := map[string]any{
		"user_input": "Hello, how are you?",
	}
	fmt.Println("(You) > " + inputs["user_input"].(string))

	result, err := exec.Execute(context.Background(), f, inputs)
	if err != nil {
		return fmt.Errorf("executing flow: %w", err)
	}

	fmt.Println("(Assistant) > " + result.Outputs["response"].(string))
	fmt.Println()
	fmt.Println("=== Execution Result ===")
	fmt.Println("Flow Name: " + f.Name)
	fmt.Println("Success: " + strconv.FormatBool(result.Success))
	fmt.Println("Duration: " + result.Duration.String())
	fmt.Println("Node Results:")
	for _, nodeResult := range result.NodeResults {
		fmt.Println("  " + nodeResult.NodeID + ": " + nodeResult.Duration.String())
	}
	return nil
}
