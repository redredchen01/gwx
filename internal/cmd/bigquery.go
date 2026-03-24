package cmd

import (
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// BigQueryCmd groups BigQuery operations.
type BigQueryCmd struct {
	Query    BQQueryCmd    `cmd:"" help:"Run a SQL query"`
	Datasets BQDatasetsCmd `cmd:"" help:"List datasets"`
	Tables   BQTablesCmd   `cmd:"" help:"List tables"`
	Describe BQDescribeCmd `cmd:"" help:"Describe a table"`
}

// resolveBQProject resolves the BigQuery project from flag or config default.
func resolveBQProject(rctx *RunContext, project string) (string, error) {
	if project != "" {
		return project, nil
	}
	val, err := config.Get("bigquery.default-project")
	if err != nil {
		return "", rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load config: %s", err))
	}
	if val == "" {
		return "", rctx.Printer.ErrExit(exitcode.UsageError,
			"project is required. Use --project or 'gwx config set bigquery.default-project <id>'")
	}
	return val, nil
}

// BQQueryCmd runs a SQL query.
type BQQueryCmd struct {
	SQL     string `arg:"" help:"SQL query to execute"`
	Project string `help:"GCP project ID" short:"p"`
	Limit   int    `help:"Max rows to return" default:"100" short:"n"`
}

func (c *BQQueryCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "bigquery.query", []string{"bigquery"}); done {
		return err
	}

	project, err := resolveBQProject(rctx, c.Project)
	if err != nil {
		return err
	}

	svc := api.NewBigQueryService(rctx.APIClient)
	result, err := svc.Query(rctx.Context, project, c.SQL, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(result)
	return nil
}

// BQDatasetsCmd lists datasets in a project.
type BQDatasetsCmd struct {
	Project string `help:"GCP project ID" short:"p"`
}

func (c *BQDatasetsCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "bigquery.datasets", []string{"bigquery"}); done {
		return err
	}

	project, err := resolveBQProject(rctx, c.Project)
	if err != nil {
		return err
	}

	svc := api.NewBigQueryService(rctx.APIClient)
	datasets, err := svc.ListDatasets(rctx.Context, project)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"project":  project,
		"datasets": datasets,
		"count":    len(datasets),
	})
	return nil
}

// BQTablesCmd lists tables in a dataset.
type BQTablesCmd struct {
	Project string `help:"GCP project ID" short:"p"`
	Dataset string `help:"Dataset ID" required:"" short:"d"`
}

func (c *BQTablesCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "bigquery.tables", []string{"bigquery"}); done {
		return err
	}

	project, err := resolveBQProject(rctx, c.Project)
	if err != nil {
		return err
	}

	svc := api.NewBigQueryService(rctx.APIClient)
	tables, err := svc.ListTables(rctx.Context, project, c.Dataset)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"project":  project,
		"dataset":  c.Dataset,
		"tables":   tables,
		"count":    len(tables),
	})
	return nil
}

// BQDescribeCmd describes a table's schema.
type BQDescribeCmd struct {
	Table   string `arg:"" help:"Table ID to describe"`
	Project string `help:"GCP project ID" short:"p"`
	Dataset string `help:"Dataset ID" required:"" short:"d"`
}

func (c *BQDescribeCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "bigquery.describe", []string{"bigquery"}); done {
		return err
	}

	project, err := resolveBQProject(rctx, c.Project)
	if err != nil {
		return err
	}

	svc := api.NewBigQueryService(rctx.APIClient)
	table, err := svc.GetTable(rctx.Context, project, c.Dataset, c.Table)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(table)
	return nil
}
