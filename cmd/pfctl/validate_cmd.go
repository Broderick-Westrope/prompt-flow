package main

import (
	"fmt"

	"github.com/broderick/prompt-flow/pkg/flow"
)

type ValidateCmd struct {
	FlowFile string `arg:"" help:"Path to flow definition file"`
}

func (c *ValidateCmd) Run() error {
	// Parse the flow
	f, err := flow.Parse(c.FlowFile)
	if err != nil {
		return fmt.Errorf("failed to parse flow: %w", err)
	}

	// Validate the flow
	if err := flow.Validate(f); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("✓ Flow '%s' is valid\n", f.Name)
	fmt.Printf("  - %d nodes\n", len(f.Nodes))
	fmt.Printf("  - Default provider: %s\n", f.Config.DefaultProvider)
	fmt.Printf("  - Default model: %s\n", f.Config.DefaultModel)

	return nil
}
