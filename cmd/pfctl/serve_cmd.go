package main

import (
	"fmt"
	"os"
	"time"

	"github.com/broderick/prompt-flow/pkg/server"
)

type ServeCmd struct {
	Port             int           `short:"p" default:"8080" help:"Port to listen on"`
	Flow             string        `arg:"" help:"Path to the flow file to serve"`
	Mode             string        `short:"m" default:"dev" enum:"dev,prod" help:"Server mode: dev (all endpoints + web UI) or prod (execute + health only)"`
	Timeout          time.Duration `short:"t" default:"5m" help:"Execution timeout for flow runs"`
	ShowStartEndNode bool          `short:"s" default:"false" help:"Show start and end nodes in the flow visualization"`
}

func (c *ServeCmd) Run() error {
	if _, err := os.Stat(c.Flow); err != nil {
		return fmt.Errorf("flow file does not exist: %w", err)
	}

	srv, err := server.New(server.Config{
		Port:             c.Port,
		FlowPath:         c.Flow,
		Mode:             c.Mode,
		ShowStartEndNode: c.ShowStartEndNode,
		ExecutionTimeout: c.Timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	return srv.Start()
}
