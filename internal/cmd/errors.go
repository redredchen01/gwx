package cmd

import (
	"errors"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
	"google.golang.org/api/googleapi"
)

// handleAPIError maps Google API errors to exit codes.
func handleAPIError(rctx *RunContext, err error) error {
	msg := err.Error()

	var circuitErr *api.CircuitOpenError
	if errors.As(err, &circuitErr) {
		return rctx.Printer.ErrExit(exitcode.CircuitOpen, msg)
	}

	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		switch gErr.Code {
		case 401:
			return rctx.Printer.ErrExit(exitcode.AuthExpired, msg)
		case 403:
			return rctx.Printer.ErrExit(exitcode.PermissionDenied, msg)
		case 404:
			return rctx.Printer.ErrExit(exitcode.NotFound, msg)
		case 429:
			return rctx.Printer.ErrExit(exitcode.RateLimited, msg)
		case 409:
			return rctx.Printer.ErrExit(exitcode.Conflict, msg)
		}
	}

	return rctx.Printer.ErrExit(exitcode.GeneralError, msg)
}
