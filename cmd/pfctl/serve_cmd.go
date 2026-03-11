package main

import (
	"fmt"
	"os"
	"time"

	"github.com/broderick/prompt-flow/pkg/server"
)

type ServeCmd struct {
	Port             int           `short:"p" default:"8080" help:"Port to listen on"`
	Flow             string        `arg:"" optional:"" help:"Path to a specific flow file to load"`
	Timeout          time.Duration `short:"t" default:"5m" help:"Execution timeout for flow runs"`
	ShowStartEndNode bool          `short:"s" default:"false" help:"Show start and end nodes in the flow visualization"`
}

func (c *ServeCmd) Run() error {
	if c.Flow != "" {
		if _, err := os.Stat(c.Flow); err != nil {
			return fmt.Errorf("flow file does not exist: %w", err)
		}
	}

	srv := server.New(server.Config{
		Port:             c.Port,
		FlowPath:         c.Flow,
		ShowStartEndNode: c.ShowStartEndNode,
		ExecutionTimeout: c.Timeout,
	})
	return srv.Start()
}
