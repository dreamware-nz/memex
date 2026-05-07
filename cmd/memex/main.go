package main

import (
	"fmt"
	"os"

	"github.com/dreamware-nz/memex/internal/adapters"
	"github.com/dreamware-nz/memex/internal/cli"
	"github.com/dreamware-nz/memex/internal/hooks"
)

var version = "0.0.0-dev"

func main() {
	root := cli.NewRootCmd()
	for _, c := range root.Commands() {
		if c.Name() == "version" {
			root.RemoveCommand(c)
			break
		}
	}
	cli.SetMCPVersion(version)
	root.AddCommand(cli.NewVersionCmd(version))
	root.AddCommand(hooks.NewHookCmd())
	root.AddCommand(adapters.NewDetectCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
