package cmd

const version = "0.24.2"

// VersionCmd prints the version.
type VersionCmd struct{}

func (c *VersionCmd) Run(rctx *RunContext) error {
	rctx.Printer.Success(map[string]string{
		"version": version,
		"name":    "gwx",
	})
	return nil
}
