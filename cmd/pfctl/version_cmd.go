package main

import "fmt"

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Printf("pfctl version %s\n", version)
	return nil
}
