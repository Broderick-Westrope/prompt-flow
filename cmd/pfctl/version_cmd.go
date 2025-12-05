package main

import "fmt"

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Printf("%s version %s\n", appName, version)
	return nil
}
