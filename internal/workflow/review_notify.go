package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/redredchen01/gwx/internal/api"
)

// ReviewNotifyResult is the output of RunReviewNotify.
type ReviewNotifyResult struct {
	Spec      string         `json:"spec"`
	Reviewers []string       `json:"reviewers"`
	Preview   string         `json:"preview"`
	Execute   *ExecuteResult `json:"execute,omitempty"`
}

// ReviewNotifyOpts configures the review-notify workflow.
type ReviewNotifyOpts struct {
	SpecFolder string
	Reviewers  []string
	Channel    string // "email" or "chat:spaces/XXX"
	Execute    bool
	NoInput    bool
	IsMCP      bool
}

// RunReviewNotify generates a review notification and optionally sends it.
func RunReviewNotify(ctx context.Context, client *api.Client, opts ReviewNotifyOpts) (*ReviewNotifyResult, error) {
	// Read sdd_context.json for spec info
	ctxPath := filepath.Join(opts.SpecFolder, "sdd_context.json")
	data, err := os.ReadFile(ctxPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ctxPath, err)
	}
	var sddCtx struct {
		SDDContext struct {
			Feature      string `json:"feature"`
			CurrentStage string `json:"current_stage"`
		} `json:"sdd_context"`
	}
	if err := json.Unmarshal(data, &sddCtx); err != nil {
		return nil, fmt.Errorf("parse sdd_context.json: %w", err)
	}

	preview := fmt.Sprintf("📋 Code Review Request: %s\nStage: %s\nReviewers: %s\nSpec: %s",
		sddCtx.SDDContext.Feature,
		sddCtx.SDDContext.CurrentStage,
		strings.Join(opts.Reviewers, ", "),
		opts.SpecFolder,
	)

	result := &ReviewNotifyResult{
		Spec:      opts.SpecFolder,
		Reviewers: opts.Reviewers,
		Preview:   preview,
	}

	// Build actions if --execute
	if opts.Execute {
		var actions []Action
		if strings.HasPrefix(opts.Channel, "chat:") {
			spaceName := strings.TrimPrefix(opts.Channel, "chat:")
			actions = append(actions, Action{
				Name:        "send_chat",
				Description: fmt.Sprintf("Send review notification to Chat space %s", spaceName),
				Fn: func(ctx context.Context) (interface{}, error) {
					svc := api.NewChatService(client)
					return svc.SendMessage(ctx, spaceName, preview)
				},
			})
		} else if opts.Channel == "email" {
			actions = append(actions, Action{
				Name:        "send_email",
				Description: fmt.Sprintf("Send review notification to %s", strings.Join(opts.Reviewers, ", ")),
				Fn: func(ctx context.Context) (interface{}, error) {
					svc := api.NewGmailService(client)
					return svc.SendMessage(ctx, &api.SendInput{
						To:      opts.Reviewers,
						Subject: fmt.Sprintf("Code Review: %s", sddCtx.SDDContext.Feature),
						Body:    preview,
					})
				},
			})
		}

		if len(actions) > 0 {
			execResult, err := Dispatch(ctx, actions, ExecuteOpts{
				Execute: opts.Execute,
				NoInput: opts.NoInput,
				IsMCP:   opts.IsMCP,
			})
			if err != nil {
				return nil, err
			}
			result.Execute = execResult
		}
	}

	return result, nil
}
