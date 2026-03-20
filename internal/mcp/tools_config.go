package mcp

import (
	"context"
	"fmt"

	"github.com/redredchen01/gwx/internal/config"
)

type configProvider struct{}

func (configProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "config_set",
			Description: "Set a configuration preference key-value pair. Persists to local preferences file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"key":   {Type: "string", Description: "Preference key (e.g. analytics.default-property)"},
					"value": {Type: "string", Description: "Value to store"},
				},
				Required: []string{"key", "value"},
			},
		},
		{
			Name:        "config_get",
			Description: "Get a single configuration preference value by key.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"key": {Type: "string", Description: "Preference key to retrieve"},
				},
				Required: []string{"key"},
			},
		},
		{
			Name:        "config_list",
			Description: "List all configuration preferences as a key-value map.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
	}
}

func (configProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"config_set": func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
			return h.configSet(args)
		},
		"config_get": func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
			return h.configGet(args)
		},
		"config_list": func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
			return h.configList()
		},
	}
}

func init() { RegisterProvider(configProvider{}) }

// --- Config handlers ---

func (h *GWXHandler) configSet(args map[string]interface{}) (*ToolResult, error) {
	key := strArg(args, "key")
	if key == "" {
		return nil, fmt.Errorf("config_set: key is required")
	}
	value := strArg(args, "value")
	if err := config.Set(key, value); err != nil {
		return nil, fmt.Errorf("config_set: %w", err)
	}
	return jsonResult(map[string]interface{}{"set": true, "key": key, "value": value})
}

func (h *GWXHandler) configGet(args map[string]interface{}) (*ToolResult, error) {
	key := strArg(args, "key")
	if key == "" {
		return nil, fmt.Errorf("config_get: key is required")
	}
	value, err := config.Get(key)
	if err != nil {
		return nil, fmt.Errorf("config_get: %w", err)
	}
	return jsonResult(map[string]interface{}{"key": key, "value": value})
}

func (h *GWXHandler) configList() (*ToolResult, error) {
	prefs, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config_list: %w", err)
	}
	return jsonResult(map[string]interface{}{"preferences": prefs, "count": len(prefs)})
}

