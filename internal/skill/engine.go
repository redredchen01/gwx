package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// MaxSkillDepth is the maximum recursion depth for skill composition (skill
// steps that invoke other skills via the "skill:<name>" tool syntax).
const MaxSkillDepth = 5

// ToolCaller abstracts MCP tool invocation so the engine does not depend on the
// mcp package directly.
type ToolCaller interface {
	CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}

// SkillLoader is a function that loads all available skills. It is used by the
// engine to resolve "skill:<name>" references without creating circular imports.
type SkillLoader func() ([]*Skill, error)

// Engine executes parsed skills against a ToolCaller.
type Engine struct {
	caller      ToolCaller
	depth       int
	skillLoader SkillLoader
}

// MaxParallelSteps limits concurrent goroutines in a parallel batch to avoid
// overwhelming API rate limits.
const MaxParallelSteps = 5

// NewEngine creates an engine that delegates tool calls to caller.
func NewEngine(caller ToolCaller) *Engine {
	return &Engine{caller: caller, depth: 0, skillLoader: CachedLoadAll}
}

// WithSkillLoader returns the engine configured with a custom skill loader.
// This is mainly useful for testing.
func (e *Engine) WithSkillLoader(loader SkillLoader) *Engine {
	e.skillLoader = loader
	return e
}

// Run executes a skill with the given input values and returns a structured result.
func (e *Engine) Run(ctx context.Context, s *Skill, inputs map[string]string) (*RunResult, error) {
	// Validate required inputs.
	for _, inp := range s.Inputs {
		if _, supplied := inputs[inp.Name]; !supplied {
			if inp.Required && inp.Default == "" {
				return nil, fmt.Errorf("missing required input %q", inp.Name)
			}
			if inp.Default != "" {
				inputs[inp.Name] = inp.Default
			}
		}
	}

	store := make(map[string]interface{}, len(s.Steps))
	result := &RunResult{
		Skill:   s.Name,
		Success: true,
		Steps:   make([]StepReport, 0, len(s.Steps)),
	}

	i := 0
	for i < len(s.Steps) {
		step := s.Steps[i]

		// Collect consecutive parallel steps into a batch.
		if step.Parallel {
			batch := collectParallelBatch(s.Steps, i)
			reports, aborted := e.runParallelBatch(ctx, batch, inputs, store)
			result.Steps = append(result.Steps, reports...)
			if aborted {
				result.Success = false
				result.Error = reports[len(reports)-1].Error
				return result, nil
			}
			i += len(batch)
			continue
		}

		// Sequential step execution.
		report, aborted := e.runSingleStep(ctx, &step, inputs, store, nil)
		result.Steps = append(result.Steps, report...)
		if aborted {
			result.Success = false
			result.Error = report[len(report)-1].Error
			return result, nil
		}
		i++
	}

	// Build final output.
	if s.Output != "" {
		rendered, err := renderValue(s.Output, inputs, store)
		if err != nil {
			result.Output = store
		} else {
			result.Output = rendered
		}
	} else {
		// Default: return last step's output.
		if len(result.Steps) > 0 {
			result.Output = result.Steps[len(result.Steps)-1].Output
		}
	}

	return result, nil
}

// collectParallelBatch returns consecutive parallel steps starting at index i.
func collectParallelBatch(steps []Step, i int) []Step {
	var batch []Step
	for j := i; j < len(steps) && steps[j].Parallel; j++ {
		batch = append(batch, steps[j])
	}
	return batch
}

// parallelResult holds the outcome of a single goroutine in a parallel batch.
type parallelResult struct {
	reports []StepReport
	aborted bool
}

// runParallelBatch executes a batch of parallel steps concurrently.
// All steps in the batch share the same snapshot of store at entry; results are
// merged back into the store after all goroutines complete.
func (e *Engine) runParallelBatch(ctx context.Context, batch []Step, inputs map[string]string, store map[string]interface{}) ([]StepReport, bool) {
	// Snapshot store so parallel steps don't race on reads.
	snapshot := make(map[string]interface{}, len(store))
	for k, v := range store {
		snapshot[k] = v
	}

	results := make([]parallelResult, len(batch))
	var wg sync.WaitGroup
	wg.Add(len(batch))

	// Semaphore limits concurrent goroutines to avoid overwhelming API rate limits.
	sem := make(chan struct{}, MaxParallelSteps)

	for idx := range batch {
		go func(i int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}: // acquire
				defer func() { <-sem }() // release
			case <-ctx.Done():
				results[i] = parallelResult{
					reports: []StepReport{{ID: batch[i].ID, Success: false, Error: ctx.Err().Error()}},
					aborted: true,
				}
				return
			}
			step := batch[i]
			// Each parallel step uses the snapshot for reads and writes to a
			// local store to avoid races.
			localStore := make(map[string]interface{}, len(snapshot))
			for k, v := range snapshot {
				localStore[k] = v
			}
			reports, aborted := e.runSingleStep(ctx, &step, inputs, localStore, nil)
			results[i] = parallelResult{
				reports: reports,
				aborted: aborted,
			}
		}(idx)
	}
	wg.Wait()

	// Merge in index order.
	var allReports []StepReport
	aborted := false

	// Build lookup map to avoid O(n²) search
	batchByID := make(map[string]Step, len(batch))
	for _, s := range batch {
		batchByID[s.ID] = s
	}

	for _, pr := range results {
		allReports = append(allReports, pr.reports...)
		if pr.aborted {
			aborted = true
		}
		// Merge results back into the main store from the reports.
		for _, r := range pr.reports {
			if r.Success {
				if s, ok := batchByID[r.ID]; ok {
					key := s.Store
					if key == "" {
						key = s.ID
					}
					store[key] = r.Output
				}
			}
		}
	}
	return allReports, aborted
}

// runSingleStep executes one step (possibly with Each iteration) and returns
// the reports. itemCtx is non-nil when called from within an Each loop to
// provide the .item namespace.
// Returns (reports, aborted).
func (e *Engine) runSingleStep(ctx context.Context, step *Step, inputs map[string]string, store map[string]interface{}, itemCtx map[string]interface{}) ([]StepReport, bool) {
	// Evaluate conditional: if the `if` field is set, render and check truthiness.
	if step.If != "" {
		condVal, err := renderValueWithItem(step.If, inputs, store, itemCtx)
		if err != nil {
			// Render error → treat as falsy, skip the step.
			return []StepReport{{
				ID:      step.ID,
				Tool:    step.Tool,
				Success: true,
				Output:  map[string]interface{}{"skipped": true, "reason": "if condition render error: " + err.Error()},
			}}, false
		}
		if !isTruthy(condVal) {
			return []StepReport{{
				ID:      step.ID,
				Tool:    step.Tool,
				Success: true,
				Output:  map[string]interface{}{"skipped": true, "reason": "if condition evaluated to false"},
			}}, false
		}
	}

	// Handle Each loop: resolve the each expression to a list and iterate.
	if step.Each != "" {
		return e.runEachStep(ctx, step, inputs, store)
	}

	report := StepReport{ID: step.ID, Tool: step.Tool}

	args, err := renderArgsWithItem(step.Args, inputs, store, itemCtx)
	if err != nil {
		report.Success = false
		report.Error = fmt.Sprintf("render args: %s", err)
		if step.OnFail == "skip" {
			return []StepReport{report}, false
		}
		return []StepReport{report}, true
	}

	// Dispatch: transform pseudo-tool, skill composition, or MCP tool call.
	var out interface{}
	if step.Tool == "transform" {
		out, err = executeTransform(args)
	} else if strings.HasPrefix(step.Tool, "skill:") {
		out, err = e.callSubSkill(ctx, step.Tool, args)
	} else {
		out, err = e.caller.CallTool(ctx, step.Tool, args)
	}

	if err != nil {
		report.Success = false
		report.Error = err.Error()
		if step.OnFail == "skip" {
			return []StepReport{report}, false
		}
		return []StepReport{report}, true
	}

	// Normalise output to map[string]interface{} for template drilling.
	normalised := normaliseOutput(out)

	report.Success = true
	report.Output = normalised

	storeKey := step.Store
	if storeKey == "" {
		storeKey = step.ID
	}
	store[storeKey] = normalised

	return []StepReport{report}, false
}

// runEachStep resolves the Each expression to a list and executes the step once
// per item. Results are collected into an array stored under the step's key.
func (e *Engine) runEachStep(ctx context.Context, step *Step, inputs map[string]string, store map[string]interface{}) ([]StepReport, bool) {
	report := StepReport{ID: step.ID, Tool: step.Tool}

	// Resolve the each expression.
	listVal, err := renderValue(step.Each, inputs, store)
	if err != nil {
		report.Success = false
		report.Error = fmt.Sprintf("resolve each: %s", err)
		if step.OnFail == "skip" {
			return []StepReport{report}, false
		}
		return []StepReport{report}, true
	}

	items := toSlice(listVal)
	if items == nil {
		report.Success = false
		report.Error = fmt.Sprintf("each: expression did not resolve to a list (got %T)", listVal)
		if step.OnFail == "skip" {
			return []StepReport{report}, false
		}
		return []StepReport{report}, true
	}

	var results []interface{}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			report.Success = false
			report.Error = err.Error()
			return []StepReport{report}, true
		}

		itemMap := toMap(item)

		// Create a step copy without Each to avoid infinite recursion.
		iterStep := *step
		iterStep.Each = ""

		reports, aborted := e.runSingleStep(ctx, &iterStep, inputs, store, itemMap)
		for _, r := range reports {
			if r.Success {
				results = append(results, r.Output)
			}
		}
		if aborted {
			// aborted on a single item — check on_fail
			if step.OnFail == "skip" {
				continue
			}
			report.Success = false
			report.Error = reports[len(reports)-1].Error
			return []StepReport{report}, true
		}
	}

	// Store the collected results.
	storeKey := step.Store
	if storeKey == "" {
		storeKey = step.ID
	}

	normalised := normaliseOutput(results)
	store[storeKey] = normalised

	report.Success = true
	report.Output = normalised
	return []StepReport{report}, false
}

// toSlice tries to convert an interface{} to []interface{}.
func toSlice(v interface{}) []interface{} {
	switch val := v.(type) {
	case []interface{}:
		return val
	case []map[string]interface{}:
		out := make([]interface{}, len(val))
		for i, m := range val {
			out[i] = m
		}
		return out
	default:
		// JSON roundtrip to try to get []interface{}.
		raw, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var arr []interface{}
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil
		}
		return arr
	}
}

// toMap tries to convert an item to map[string]interface{} for the .item namespace.
func toMap(v interface{}) map[string]interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return val
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return map[string]interface{}{"value": v}
		}
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return map[string]interface{}{"value": v}
		}
		return m
	}
}

// renderArgsWithItem extends renderArgs with optional .item context for Each loops.
func renderArgsWithItem(raw map[string]string, inputs map[string]string, store map[string]interface{}, itemCtx map[string]interface{}) (map[string]interface{}, error) {
	if itemCtx == nil {
		return renderArgs(raw, inputs, store)
	}
	out := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		resolved, err := renderValueWithItem(v, inputs, store, itemCtx)
		if err != nil {
			return nil, fmt.Errorf("arg %q: %w", k, err)
		}
		out[k] = resolved
	}
	return out, nil
}

// executeTransform applies in-process data transformations without calling MCP.
func executeTransform(args map[string]interface{}) (interface{}, error) {
	input, ok := args["input"]
	if !ok {
		return nil, fmt.Errorf("transform: 'input' argument is required")
	}

	result := input

	// pick: keep only named fields from each item in a list.
	if pickStr, ok := args["pick"].(string); ok && pickStr != "" {
		fields := strings.Split(pickStr, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		result = transformPick(result, fields)
	}

	// flatten: flatten array of arrays.
	if _, ok := args["flatten"]; ok {
		result = transformFlatten(result)
	}

	// sort_by: sort by field.
	if sortField, ok := args["sort_by"].(string); ok && sortField != "" {
		result = transformSortBy(result, sortField)
	}

	// limit: take first N items.
	if limitVal, ok := args["limit"]; ok {
		n, err := toInt(limitVal)
		if err != nil {
			return nil, fmt.Errorf("transform: limit: %w", err)
		}
		result = transformLimit(result, n)
	}

	// count: return length of input.
	if _, ok := args["count"]; ok {
		result = transformCount(result)
	}

	return result, nil
}

func transformPick(data interface{}, fields []string) interface{} {
	items := toSlice(data)
	if items == nil {
		// Single item: treat as one-element list.
		m, ok := data.(map[string]interface{})
		if !ok {
			return data
		}
		picked := make(map[string]interface{}, len(fields))
		for _, f := range fields {
			if v, exists := m[f]; exists {
				picked[f] = v
			}
		}
		return picked
	}

	result := make([]interface{}, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			result = append(result, item)
			continue
		}
		picked := make(map[string]interface{}, len(fields))
		for _, f := range fields {
			if v, exists := m[f]; exists {
				picked[f] = v
			}
		}
		result = append(result, picked)
	}
	return result
}

func transformFlatten(data interface{}) interface{} {
	items := toSlice(data)
	if items == nil {
		return data
	}
	var flat []interface{}
	for _, item := range items {
		sub := toSlice(item)
		if sub != nil {
			flat = append(flat, sub...)
		} else {
			flat = append(flat, item)
		}
	}
	return flat
}

func transformSortBy(data interface{}, field string) interface{} {
	items := toSlice(data)
	if items == nil {
		return data
	}
	sorted := make([]interface{}, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		mi, _ := sorted[i].(map[string]interface{})
		mj, _ := sorted[j].(map[string]interface{})
		if mi == nil || mj == nil {
			return false
		}
		vi := fmt.Sprintf("%v", mi[field])
		vj := fmt.Sprintf("%v", mj[field])
		return vi < vj
	})
	return sorted
}

func transformLimit(data interface{}, n int) interface{} {
	items := toSlice(data)
	if items == nil {
		return data
	}
	if n >= len(items) {
		return items
	}
	if n < 0 {
		n = 0
	}
	return items[:n]
}

func transformCount(data interface{}) interface{} {
	items := toSlice(data)
	if items == nil {
		return float64(0)
	}
	return float64(len(items))
}

func toInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case float64:
		return int(val), nil
	case string:
		return strconv.Atoi(val)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

// callSubSkill resolves a "skill:<name>" tool reference, loads the target
// skill, and runs it as a sub-engine with incremented recursion depth.
func (e *Engine) callSubSkill(ctx context.Context, tool string, args map[string]interface{}) (interface{}, error) {
	skillName := strings.TrimPrefix(tool, "skill:")
	if skillName == "" {
		return nil, fmt.Errorf("skill: empty skill name in tool reference %q", tool)
	}

	if e.depth+1 > MaxSkillDepth {
		return nil, fmt.Errorf("skill composition depth limit (%d) exceeded when calling %q — possible infinite recursion", MaxSkillDepth, skillName)
	}

	loader := e.skillLoader
	if loader == nil {
		loader = LoadAll
	}

	skills, err := loader()
	if err != nil {
		return nil, fmt.Errorf("load skills for composition: %w", err)
	}

	var target *Skill
	for _, s := range skills {
		if s.Name == skillName {
			target = s
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("skill %q not found (referenced by skill:%s)", skillName, skillName)
	}

	subEngine := &Engine{
		caller:      e.caller,
		depth:       e.depth + 1,
		skillLoader: e.skillLoader,
	}

	subInputs := argsToInputs(args)
	result, err := subEngine.Run(ctx, target, subInputs)
	if err != nil {
		return nil, fmt.Errorf("sub-skill %q: %w", skillName, err)
	}
	if !result.Success {
		return nil, fmt.Errorf("sub-skill %q failed: %s", skillName, result.Error)
	}
	return result.Output, nil
}

// argsToInputs converts map[string]interface{} (tool call args) to
// map[string]string (skill inputs).
func argsToInputs(args map[string]interface{}) map[string]string {
	inputs := make(map[string]string, len(args))
	for k, v := range args {
		inputs[k] = fmt.Sprintf("%v", v)
	}
	return inputs
}

// isTruthy evaluates whether a value is considered truthy for conditional step
// execution. Falsy values: nil, empty string, "false", "0", float64(0), bool false,
// empty slice, empty map. Everything else is truthy.
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "false" && val != "0"
	case float64:
		return val != 0
	case int:
		return val != 0
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	default:
		// For any other type, consider it truthy if non-nil (already checked above).
		return true
	}
}

// normaliseOutput tries to unwrap the output into a map for template drilling.
// If the output is a JSON string, it is parsed. Otherwise it's left as-is.
func normaliseOutput(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return val
	case []interface{}:
		return val
	case string:
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(val), &m); err == nil {
			return m
		}
		return val
	default:
		// JSON roundtrip to get map[string]interface{}.
		raw, err := json.Marshal(val)
		if err != nil {
			return val
		}
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			// Try as array.
			var arr []interface{}
			if err2 := json.Unmarshal(raw, &arr); err2 == nil {
				return arr
			}
			return val
		}
		return m
	}
}
