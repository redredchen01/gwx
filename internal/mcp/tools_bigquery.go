package mcp

import (
	"context"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

type bigqueryProvider struct{}

func (bigqueryProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "bigquery_query",
			Description: "Run a BigQuery SQL query (Standard SQL). Returns column headers and result rows. Uses synchronous jobs.query.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project": {Type: "string", Description: "GCP project ID. If omitted, uses default from config."},
					"sql":     {Type: "string", Description: "SQL query to execute (Standard SQL)"},
					"limit":   {Type: "integer", Description: "Max rows to return (default 100)"},
				},
				Required: []string{"sql"},
			},
		},
		{
			Name:        "bigquery_datasets",
			Description: "List all datasets in a BigQuery project.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project": {Type: "string", Description: "GCP project ID. If omitted, uses default from config."},
				},
			},
		},
		{
			Name:        "bigquery_tables",
			Description: "List all tables in a BigQuery dataset.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project": {Type: "string", Description: "GCP project ID. If omitted, uses default from config."},
					"dataset": {Type: "string", Description: "Dataset ID"},
				},
				Required: []string{"dataset"},
			},
		},
		{
			Name:        "bigquery_describe",
			Description: "Describe a BigQuery table's schema, row count, size, and partitioning.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project":  {Type: "string", Description: "GCP project ID. If omitted, uses default from config."},
					"dataset":  {Type: "string", Description: "Dataset ID"},
					"table_id": {Type: "string", Description: "Table ID"},
				},
				Required: []string{"dataset", "table_id"},
			},
		},
	}
}

func (bigqueryProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"bigquery_query":    h.bigqueryQuery,
		"bigquery_datasets": h.bigqueryDatasets,
		"bigquery_tables":   h.bigqueryTables,
		"bigquery_describe": h.bigqueryDescribe,
	}
}

func init() { RegisterProvider(bigqueryProvider{}) }

// --- helpers ---

// resolveBQProjectMCP returns the project arg if provided, otherwise falls back to config.
func resolveBQProjectMCP(args map[string]interface{}) (string, error) {
	if p := strArg(args, "project"); p != "" {
		return p, nil
	}
	p, err := config.Get("bigquery.default-project")
	if err != nil {
		return "", fmt.Errorf("bigquery: could not read default project from config: %w", err)
	}
	if p == "" {
		return "", fmt.Errorf("bigquery: project not provided and bigquery.default-project is not configured. Provide 'project' parameter or run: gwx config set bigquery.default-project <id>")
	}
	return p, nil
}

// --- BigQuery handlers ---

func (h *GWXHandler) bigqueryQuery(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	project, err := resolveBQProjectMCP(args)
	if err != nil {
		return nil, err
	}

	svc := api.NewBigQueryService(h.client)
	result, err := svc.Query(ctx, project, strArg(args, "sql"), intArg(args, "limit", 100))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) bigqueryDatasets(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	project, err := resolveBQProjectMCP(args)
	if err != nil {
		return nil, err
	}

	svc := api.NewBigQueryService(h.client)
	datasets, err := svc.ListDatasets(ctx, project)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{
		"project":  project,
		"datasets": datasets,
		"count":    len(datasets),
	})
}

func (h *GWXHandler) bigqueryTables(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	project, err := resolveBQProjectMCP(args)
	if err != nil {
		return nil, err
	}

	dataset := strArg(args, "dataset")
	svc := api.NewBigQueryService(h.client)
	tables, err := svc.ListTables(ctx, project, dataset)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{
		"project": project,
		"dataset": dataset,
		"tables":  tables,
		"count":   len(tables),
	})
}

func (h *GWXHandler) bigqueryDescribe(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	project, err := resolveBQProjectMCP(args)
	if err != nil {
		return nil, err
	}

	svc := api.NewBigQueryService(h.client)
	table, err := svc.GetTable(ctx, project, strArg(args, "dataset"), strArg(args, "table_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(table)
}
