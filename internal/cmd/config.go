package cmd

import (
	"fmt"
	"sort"

	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// ConfigCmd groups configuration management operations.
type ConfigCmd struct {
	Set  ConfigSetCmd  `cmd:"" help:"Set a configuration value"`
	Get  ConfigGetCmd  `cmd:"" help:"Get a configuration value"`
	List ConfigListCmd `cmd:"" help:"List all configuration values"`
}

// ConfigSetCmd sets a single configuration key.
type ConfigSetCmd struct {
	Key   string `arg:"" help:"Configuration key (e.g. analytics.default-property)"`
	Value string `arg:"" help:"Configuration value"`
}

func (c *ConfigSetCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "config.set"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{
			"dry_run": "config.set",
			"key":     c.Key,
			"value":   c.Value,
		})
		return nil
	}
	if err := config.Set(c.Key, c.Value); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("set config: %s", err))
	}
	rctx.Printer.Success(map[string]string{"set": c.Key, "value": c.Value})
	return nil
}

// ConfigGetCmd retrieves a single configuration key.
type ConfigGetCmd struct {
	Key string `arg:"" help:"Configuration key to retrieve"`
}

func (c *ConfigGetCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "config.get"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "config.get", "key": c.Key})
		return nil
	}
	val, err := config.Get(c.Key)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("get config: %s", err))
	}
	rctx.Printer.Success(map[string]string{"key": c.Key, "value": val})
	return nil
}

// ConfigListCmd lists all configuration key-value pairs.
type ConfigListCmd struct{}

func (c *ConfigListCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "config.list"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "config.list"})
		return nil
	}
	prefs, err := config.Load()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load config: %s", err))
	}

	// Stable output: sort keys.
	keys := make([]string, 0, len(prefs))
	for k := range prefs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	items := make([]map[string]string, 0, len(prefs))
	for _, k := range keys {
		items = append(items, map[string]string{"key": k, "value": prefs[k]})
	}
	rctx.Printer.Success(map[string]interface{}{"preferences": items, "count": len(items)})
	return nil
}
