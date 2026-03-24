package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// FormsCmd groups Google Forms operations.
type FormsCmd struct {
	Get       FormsGetCmd       `cmd:"" help:"Get form structure"`
	Responses FormsResponsesCmd `cmd:"" help:"List form responses"`
	Response  FormsResponseCmd  `cmd:"" help:"Get a single form response"`
}

// FormsGetCmd retrieves a form's structure.
type FormsGetCmd struct {
	FormID string `arg:"" help:"Form ID to retrieve"`
}

func (c *FormsGetCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "forms.get", []string{"forms"}); done {
		return err
	}

	svc := api.NewFormsService(rctx.APIClient)
	form, err := svc.GetForm(rctx.Context, c.FormID)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(form)
	return nil
}

// FormsResponsesCmd lists form responses.
type FormsResponsesCmd struct {
	FormID string `arg:"" help:"Form ID"`
	Limit  int    `help:"Max responses to return" default:"50" short:"n"`
}

func (c *FormsResponsesCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "forms.responses", []string{"forms"}); done {
		return err
	}

	svc := api.NewFormsService(rctx.APIClient)
	responses, err := svc.ListResponses(rctx.Context, c.FormID, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"form_id":   c.FormID,
		"responses": responses,
		"count":     len(responses),
	})
	return nil
}

// FormsResponseCmd gets a single form response.
type FormsResponseCmd struct {
	FormID     string `arg:"" help:"Form ID"`
	ResponseID string `arg:"" help:"Response ID"`
}

func (c *FormsResponseCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "forms.response", []string{"forms"}); done {
		return err
	}

	svc := api.NewFormsService(rctx.APIClient)

	if c.ResponseID == "" {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, "response ID is required")
	}

	resp, err := svc.GetResponse(rctx.Context, c.FormID, c.ResponseID)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(resp)
	return nil
}
