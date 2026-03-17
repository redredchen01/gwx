package cmd

import "fmt"

const version = "0.1.0"

// VersionCmd prints the version.
type VersionCmd struct{}

func (c *VersionCmd) Run(rctx *RunContext) error {
	rctx.Printer.Success(map[string]string{
		"version": version,
		"name":    "gwx",
	})
	fmt.Fprintf(rctx.Printer.Writer, "")
	return nil
}
