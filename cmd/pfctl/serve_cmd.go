package main

import (
	"fmt"
	"os"

	"github.com/broderick/prompt-flow/pkg/server"
)

type ServeCmd struct {
	Port int    `short:"p" default:"8080" help:"Port to listen on"`
	Flow string `arg:"" help:"Path to a specific flow file to load"`
}

func (c *ServeCmd) Run() error {
	if _, err := os.Stat(c.Flow); err != nil {
		return fmt.Errorf("flow file does not exist: %w", err)
	}

	srv := server.New(c.Port, c.Flow)

	fmt.Printf("Starting prompt flow web UI on http://localhost:%d\n", c.Port)
	fmt.Println("Press Ctrl+C to stop")

	return srv.Start()
}
