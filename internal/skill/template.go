package skill

import (
	"encoding/json"
	"fmt"
	"strings"
)

// renderArgs interpolates template expressions in step args.
//
// Supported syntax:
//
//	{{.input.name}}        — user-supplied input value
//	{{.steps.id.field}}    — output field from a previous step
//	{{.steps.id}}          — full JSON output of a previous step
//	{{.item.field}}        — field from current Each iteration item
func renderArgs(raw map[string]string, inputs map[string]string, store map[string]interface{}) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		resolved, err := renderValue(v, inputs, store)
		if err != nil {
			return nil, fmt.Errorf("arg %q: %w", k, err)
		}
		out[k] = resolved
	}
	return out, nil
}

func renderValue(tmpl string, inputs map[string]string, store map[string]interface{}) (interface{}, error) {
	return renderValueWithItem(tmpl, inputs, store, nil)
}

// renderValueWithItem is the extended version of renderValue that supports the
// .item namespace for Each loop iterations.
func renderValueWithItem(tmpl string, inputs map[string]string, store map[string]interface{}, itemCtx map[string]interface{}) (interface{}, error) {
	// Fast path: no template syntax
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}

	// If the entire value is a single template expression, return the native type.
	trimmed := strings.TrimSpace(tmpl)
	if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") && strings.Count(trimmed, "{{") == 1 {
		expr := strings.TrimSpace(trimmed[2 : len(trimmed)-2])
		val, err := resolveExprWithItem(expr, inputs, store, itemCtx)
		if err != nil {
			return nil, err
		}
		return val, nil
	}

	// Mixed template: interpolate all {{...}} as strings using a builder
	// to avoid O(n^2) string concatenation.
	var b strings.Builder
	b.Grow(len(tmpl))
	pos := 0
	for {
		start := strings.Index(tmpl[pos:], "{{")
		if start == -1 {
			b.WriteString(tmpl[pos:])
			break
		}
		start += pos
		b.WriteString(tmpl[pos:start])
		end := strings.Index(tmpl[start:], "}}")
		if end == -1 {
			return nil, fmt.Errorf("unclosed template expression in %q", tmpl)
		}
		end += start + 2
		expr := strings.TrimSpace(tmpl[start+2 : end-2])
		val, err := resolveExprWithItem(expr, inputs, store, itemCtx)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&b, "%v", val)
		pos = end
	}
	return b.String(), nil
}

// resolveExprWithItem resolves a template expression with optional .item namespace.
func resolveExprWithItem(expr string, inputs map[string]string, store map[string]interface{}, itemCtx map[string]interface{}) (interface{}, error) {
	// Strip leading dot: ".input.name" → "input.name"
	expr = strings.TrimPrefix(expr, ".")

	parts := strings.SplitN(expr, ".", 3)
	if len(parts) < 2 {
		// Allow ".item" alone (without sub-field) to return the whole item map.
		if parts[0] == "item" && itemCtx != nil {
			return itemCtx, nil
		}
		return nil, fmt.Errorf("invalid expression: %q", expr)
	}

	switch parts[0] {
	case "input":
		v, ok := inputs[parts[1]]
		if !ok {
			return nil, fmt.Errorf("unknown input %q", parts[1])
		}
		return v, nil

	case "steps":
		stepID := parts[1]
		stepOut, ok := store[stepID]
		if !ok {
			return nil, fmt.Errorf("no output from step %q", stepID)
		}
		if len(parts) == 2 {
			return stepOut, nil
		}
		// Drill into step output (expects map)
		return drillField(stepOut, parts[2])

	case "item":
		if itemCtx == nil {
			return nil, fmt.Errorf("'.item' is only available inside an 'each' loop")
		}
		field := parts[1]
		if len(parts) == 3 {
			field = parts[1] + "." + parts[2]
		}
		return drillField(itemCtx, field)

	default:
		return nil, fmt.Errorf("unknown namespace %q in expression", parts[0])
	}
}

// drillField navigates dotted paths like "messages.0.subject" into a nested structure.
func drillField(data interface{}, path string) (interface{}, error) {
	// Normalise to map[string]interface{} via JSON roundtrip if needed.
	m, ok := data.(map[string]interface{})
	if !ok {
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("cannot drill into non-map output: %T", data)
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("cannot drill into non-map output: %T", data)
		}
	}

	segments := strings.Split(path, ".")
	var current interface{} = m
	for _, seg := range segments {
		switch cur := current.(type) {
		case map[string]interface{}:
			v, exists := cur[seg]
			if !exists {
				return nil, fmt.Errorf("field %q not found in step output", seg)
			}
			current = v
		default:
			return nil, fmt.Errorf("cannot navigate into %T with key %q", current, seg)
		}
	}
	return current, nil
}
