package cmd

import (
	"github.com/user/gwx/internal/exitcode"
)

// AgentCmd provides helpers for LLM agent integration.
type AgentCmd struct {
	ExitCodes AgentExitCodesCmd `cmd:"exit-codes" help:"Print stable exit code reference"`
}

// AgentExitCodesCmd prints all exit codes for agent automation.
type AgentExitCodesCmd struct{}

func (c *AgentExitCodesCmd) Run(rctx *RunContext) error {
	codes := exitcode.All()
	rctx.Printer.Success(codes)
	return nil
}
