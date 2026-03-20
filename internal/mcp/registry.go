package mcp

import "sync"

// ToolProvider is implemented by each tools_*.go to self-register tools and handlers.
type ToolProvider interface {
	Tools() []Tool
	Handlers(h *GWXHandler) map[string]ToolHandler
}

// ToolRegistry maintains the ordered list of registered ToolProviders.
type ToolRegistry struct {
	mu        sync.RWMutex
	providers []ToolProvider
}

var globalRegistry = &ToolRegistry{}

// RegisterProvider adds a ToolProvider to the global registry.
func RegisterProvider(p ToolProvider) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.providers = append(globalRegistry.providers, p)
}

func (r *ToolRegistry) allTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var tools []Tool
	for _, p := range r.providers {
		tools = append(tools, p.Tools()...)
	}
	return tools
}

func (r *ToolRegistry) buildHandlers(h *GWXHandler) map[string]ToolHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]ToolHandler)
	for _, p := range r.providers {
		for k, v := range p.Handlers(h) {
			if _, exists := result[k]; exists {
				panic("mcp: duplicate tool key registered: " + k)
			}
			result[k] = v
		}
	}
	return result
}
