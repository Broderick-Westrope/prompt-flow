package main

import (
	"fmt"

	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/fatih/color"
)

type ValidateCmd struct {
	FlowFile string `arg:"" help:"Path to flow definition file"`
	Plain    bool   `help:"Disable color output"`
}

func (c *ValidateCmd) Run() error {
	f, err := flow.Parse(c.FlowFile)
	if err != nil {
		return fmt.Errorf("failed to parse flow: %w", err)
	}

	err = flow.Validate(f)
	if err != nil {
		red := noColorIfFlagSet(c.Plain, color.FgRed)
		fmt.Printf("%s Flow is %s:\n", red("✗"), red("invalid"))
		fmt.Printf("%s\n", err)
		return nil
	}

	green := noColorIfFlagSet(c.Plain, color.FgGreen)
	fmt.Printf("%s Flow '%s' is %s\n", green("✓"), f.Name, green("valid"))
	fmt.Printf("  - %d nodes\n", len(f.Nodes))
	fmt.Printf("  - Default provider: %s\n", f.Config.DefaultProvider)
	fmt.Printf("  - Default model: %s\n", f.Config.DefaultModel)

	return nil
}
