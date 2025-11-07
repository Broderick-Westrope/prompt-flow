package main

import (
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
