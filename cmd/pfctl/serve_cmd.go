package main

import (
	"fmt"
	"os"
	"time"

	"github.com/broderick/prompt-flow/pkg/server"
)

type ServeCmd struct {
	Port             int           `short:"p" default:"8080" help:"Port to listen on"`
	Flow             string        `arg:"" help:"Path to a specific flow file to load"`
	ShowStartEndNode bool          `short:"s" default:"false" help:"Show start and end nodes in the flow visualization"`
	Timeout          time.Duration `short:"t" default:"5m" help:"Execution timeout for flow runs"`
}

func (c *ServeCmd) Run() error {
	if _, err := os.Stat(c.Flow); err != nil {
		return fmt.Errorf("flow file does not exist: %w", err)
	}

	srv := server.New(c.Port, c.Flow, c.ShowStartEndNode, c.Timeout)
	return srv.Start()
}
