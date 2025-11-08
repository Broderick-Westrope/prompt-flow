package main

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/broderick/prompt-flow/pkg/flow"
)

type InitCmd struct {
	Name   string `arg:"" help:"Name of the flow"`
	Output string `short:"o" help:"Output file path (default: [name].flow.yaml)"`
	Format string `short:"f" default:"yaml" enum:"yaml,json" help:"Output format: yaml or json"`
}

func (c *InitCmd) Run() error {
	outputPath := c.Output
	if outputPath == "" {
		if c.Format == "json" {
			outputPath = fmt.Sprintf("%s.flow.json", c.Name)
		} else {
			outputPath = fmt.Sprintf("%s.flow.yaml", c.Name)
		}
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("file already exists: %s", outputPath)
	}

	// Create sample flow
	sampleFlow := createSampleFlow(c.Name)

	// Save the flow
	if err := flow.Save(sampleFlow, outputPath); err != nil {
		return fmt.Errorf("failed to save flow: %w", err)
	}

	fmt.Printf("Created flow definition: %s\n", outputPath)
	return nil
}

func createSampleFlow(name string) *flow.Flow {
	return &flow.Flow{
		Version:     "1.0",
		Name:        name,
		Description: "A sample prompt flow",
		Config: flow.Config{
			DefaultProvider: "github_playground_openai",
			DefaultModel:    "openai/gpt-4o-mini",
		},
		Nodes: []flow.Node{
			{
				ID:       "process",
				Provider: "github_playground_openai",
				Model:    "openai/gpt-4o-mini",
				Inputs: []flow.Input{
					{
						Name: "user_input",
						From: "input",
					},
				},
				Prompt: heredoc.Doc(`
					You are a helpful assistant.
					
					User input: {{.user_input}}

					Please provide a helpful response.
				`),
				Outputs: []flow.Output{
					{
						Name: "response",
						To:   "output",
					},
				},
			},
		},
	}
}
