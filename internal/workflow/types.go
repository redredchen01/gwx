// Package workflow provides composable workflow implementations for gwx.
//
// Each workflow exposes a top-level function matching the canonical signature:
//
//	func RunXxx(ctx context.Context, client *api.Client, opts XxxOpts) (*XxxResult, error)
//
// Where XxxOpts contains all input parameters and *XxxResult is the structured output.
// Some workflows use the aggregator.Fetcher pattern for parallel data fetching,
// while action-oriented workflows use the executor.Action pattern.
package workflow

import (
	"context"

	"github.com/redredchen01/gwx/internal/api"
)

// Runnable is implemented by workflow adapter structs that need dynamic dispatch.
// Used by cmd/workflow.go to dispatch to the correct workflow without type switching.
//
// Example adapter:
//
//	type standupAdapter struct{ opts StandupOpts }
//	func (a standupAdapter) Run(ctx context.Context, client *api.Client) (interface{}, error) {
//	    return RunStandup(ctx, client, a.opts)
//	}
type Runnable interface {
	Run(ctx context.Context, client *api.Client) (interface{}, error)
}

// WorkflowFunc documents the canonical function signature for all workflow entry points.
// This is a documentation type — Go interfaces cannot have generic methods,
// so this type is not used for runtime dispatch. Use Runnable for that.
type WorkflowFunc[O any, R any] func(ctx context.Context, client *api.Client, opts O) (*R, error)
