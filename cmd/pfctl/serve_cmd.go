package main

import (
	"fmt"
	"os"

	"github.com/broderick/prompt-flow/pkg/server"
)

type ServeCmd struct {
	Port             int    `short:"p" default:"8080" help:"Port to listen on"`
	Flow             string `arg:"" help:"Path to a specific flow file to load"`
	ShowStartEndNode bool   `short:"s" default:"false" help:"Show start and end nodes in the flow visualization"`
}

func (c *ServeCmd) Run() error {
	if _, err := os.Stat(c.Flow); err != nil {
		return fmt.Errorf("flow file does not exist: %w", err)
	}

	srv := server.New(c.Port, c.Flow, c.ShowStartEndNode)

	fmt.Printf("Starting prompt flow web UI on http://localhost:%d\n", c.Port)
	fmt.Println("Press Ctrl+C to stop")

	return srv.Start()
}
