package main

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/alecthomas/kong"
	"github.com/broderick/prompt-flow/pkg/flow"
)

var (
	version = "0.1.0"
)

type CLI struct {
	Init     InitCmd     `cmd:"" help:"Initialize a new prompt flow"`
	Validate ValidateCmd `cmd:"" help:"Validate a prompt flow definition"`
	Test     TestCmd     `cmd:"" help:"Test a prompt flow with sample inputs"`
	Serve    ServeCmd    `cmd:"" help:"Start the web UI server"`
	Version  VersionCmd  `cmd:"" help:"Show version information"`
}

func main() {
	cli := &CLI{}

	ctx := kong.Parse(cli,
		kong.Name("pfctl"),
		kong.Description("Prompt Flow Control - A tool for managing prompt flows"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

func createSampleFlow(name string) *flow.Flow {
	return &flow.Flow{
		Version:     "1.0",
		Name:        name,
		Description: "A sample prompt flow",
		Config: flow.Config{
			DefaultProvider: "openai",
			DefaultModel:    "gpt-3.5-turbo",
		},
		Nodes: []flow.Node{
			{
				ID:       "process",
				Type:     "llm",
				Provider: "openai",
				Model:    "gpt-3.5-turbo",
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

	for i, nodeResult := range result.NodeResults {
		fmt.Printf("\n[%d] Node: %s\n", i+1, nodeResult.NodeID)
		fmt.Printf("    Success: %v\n", nodeResult.Success)
		fmt.Printf("    Duration: %v\n", nodeResult.Duration)

		if nodeResult.Error != "" {
			fmt.Printf("    Error: %s\n", nodeResult.Error)
		}

		if nodeResult.Metrics.TokensUsed > 0 {
			fmt.Printf("    Tokens: %d (prompt: %d, completion: %d)\n",
				nodeResult.Metrics.TokensUsed,
				nodeResult.Metrics.PromptTokens,
				nodeResult.Metrics.CompletionTokens)
			fmt.Printf("    Estimated Cost: $%.6f\n", nodeResult.Metrics.EstimatedCost)

			totalTokens += nodeResult.Metrics.TokensUsed
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

	if len(result.Outputs) > 0 {
		fmt.Printf("\n=== Flow Outputs ===\n")
		outputJSON, _ := json.MarshalIndent(result.Outputs, "", "  ")
		fmt.Println(string(outputJSON))
	}
}
