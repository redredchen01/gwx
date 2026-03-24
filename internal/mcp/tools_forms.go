package mcp

import (
	"context"

	"github.com/redredchen01/gwx/internal/api"
)

type formsProvider struct{}

func (formsProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "forms_get",
			Description: "Get the structure of a Google Form (title, questions, options). Returns form metadata and all items.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"form_id": {Type: "string", Description: "Google Form ID"},
				},
				Required: []string{"form_id"},
			},
		},
		{
			Name:        "forms_responses",
			Description: "List responses submitted to a Google Form. Returns respondent answers, timestamps, and optional email.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"form_id": {Type: "string", Description: "Google Form ID"},
					"limit":   {Type: "integer", Description: "Max responses to return (default 50)"},
				},
				Required: []string{"form_id"},
			},
		},
		{
			Name:        "forms_response",
			Description: "Get a single response from a Google Form by response ID.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"form_id":     {Type: "string", Description: "Google Form ID"},
					"response_id": {Type: "string", Description: "Response ID to retrieve"},
				},
				Required: []string{"form_id", "response_id"},
			},
		},
	}
}

func (formsProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"forms_get":      h.formsGet,
		"forms_responses": h.formsResponses,
		"forms_response":  h.formsResponse,
	}
}

func init() { RegisterProvider(formsProvider{}) }

// --- Forms handlers ---

func (h *GWXHandler) formsGet(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewFormsService(h.client)
	form, err := svc.GetForm(ctx, strArg(args, "form_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(form)
}

func (h *GWXHandler) formsResponses(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewFormsService(h.client)
	responses, err := svc.ListResponses(ctx, strArg(args, "form_id"), intArg(args, "limit", 50))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{
		"form_id":   strArg(args, "form_id"),
		"responses": responses,
		"count":     len(responses),
	})
}

func (h *GWXHandler) formsResponse(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewFormsService(h.client)
	resp, err := svc.GetResponse(ctx, strArg(args, "form_id"), strArg(args, "response_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(resp)
}
