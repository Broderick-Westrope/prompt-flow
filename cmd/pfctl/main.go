package main

import (
	"github.com/alecthomas/kong"
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
