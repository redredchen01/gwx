package api

import (
	"context"
	"fmt"

	bigquery "google.golang.org/api/bigquery/v2"
)

// BigQueryService wraps Google BigQuery API operations.
type BigQueryService struct {
	client *Client
}

// NewBigQueryService creates a BigQuery service wrapper.
func NewBigQueryService(client *Client) *BigQueryService {
	return &BigQueryService{client: client}
}

// Query executes a synchronous SQL query using jobs.query and returns parsed results.
func (bq *BigQueryService) Query(ctx context.Context, projectID, sql string, limit int) (map[string]interface{}, error) {
	opts, err := bq.client.ServiceInit(ctx, "bigquery")
	if err != nil {
		return nil, err
	}

	svc, err := bigquery.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create bigquery service: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}

	useLegacy := false
	req := &bigquery.QueryRequest{
		Query:        sql,
		UseLegacySql: &useLegacy,
		MaxResults:   int64(limit),
	}

	resp, err := svc.Jobs.Query(projectID, req).Do()
	if err != nil {
		return nil, fmt.Errorf("bigquery query project %s: %w", projectID, err)
	}

	// Extract column headers.
	var columns []string
	for _, field := range resp.Schema.Fields {
		columns = append(columns, field.Name)
	}

	// Parse rows into maps keyed by column name.
	var rows []map[string]interface{}
	for _, row := range resp.Rows {
		r := make(map[string]interface{}, len(columns))
		for i, cell := range row.F {
			if i < len(columns) {
				r[columns[i]] = cell.V
			}
		}
		rows = append(rows, r)
	}

	result := map[string]interface{}{
		"project":       projectID,
		"columns":       columns,
		"rows":          rows,
		"row_count":     len(rows),
		"total_rows":    resp.TotalRows,
		"job_complete":  resp.JobComplete,
		"cache_hit":     resp.CacheHit,
	}

	if resp.TotalBytesProcessed > 0 {
		result["total_bytes_processed"] = resp.TotalBytesProcessed
	}

	return result, nil
}

// ListDatasets lists all datasets in a BigQuery project.
func (bq *BigQueryService) ListDatasets(ctx context.Context, projectID string) ([]map[string]interface{}, error) {
	opts, err := bq.client.ServiceInit(ctx, "bigquery")
	if err != nil {
		return nil, err
	}

	svc, err := bigquery.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create bigquery service: %w", err)
	}

	var datasets []map[string]interface{}
	pageToken := ""

	for {
		call := svc.Datasets.List(projectID)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("bigquery list datasets project %s: %w", projectID, err)
		}

		for _, ds := range resp.Datasets {
			entry := map[string]interface{}{
				"dataset_id":    ds.DatasetReference.DatasetId,
				"project":       ds.DatasetReference.ProjectId,
				"friendly_name": ds.FriendlyName,
				"location":      ds.Location,
			}
			if ds.Labels != nil {
				entry["labels"] = ds.Labels
			}
			datasets = append(datasets, entry)
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return datasets, nil
}

// ListTables lists all tables in a BigQuery dataset.
func (bq *BigQueryService) ListTables(ctx context.Context, projectID, datasetID string) ([]map[string]interface{}, error) {
	opts, err := bq.client.ServiceInit(ctx, "bigquery")
	if err != nil {
		return nil, err
	}

	svc, err := bigquery.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create bigquery service: %w", err)
	}

	var tables []map[string]interface{}
	pageToken := ""

	for {
		call := svc.Tables.List(projectID, datasetID)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("bigquery list tables %s.%s: %w", projectID, datasetID, err)
		}

		for _, t := range resp.Tables {
			entry := map[string]interface{}{
				"table_id":   t.TableReference.TableId,
				"dataset_id": t.TableReference.DatasetId,
				"project":    t.TableReference.ProjectId,
				"type":       t.Type,
			}
			if t.FriendlyName != "" {
				entry["friendly_name"] = t.FriendlyName
			}
			if t.TimePartitioning != nil {
				entry["partitioning"] = map[string]interface{}{
					"type":  t.TimePartitioning.Type,
					"field": t.TimePartitioning.Field,
				}
			}
			if t.Labels != nil {
				entry["labels"] = t.Labels
			}
			tables = append(tables, entry)
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return tables, nil
}

// GetTable retrieves metadata for a specific BigQuery table.
func (bq *BigQueryService) GetTable(ctx context.Context, projectID, datasetID, tableID string) (map[string]interface{}, error) {
	opts, err := bq.client.ServiceInit(ctx, "bigquery")
	if err != nil {
		return nil, err
	}

	svc, err := bigquery.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create bigquery service: %w", err)
	}

	table, err := svc.Tables.Get(projectID, datasetID, tableID).Do()
	if err != nil {
		return nil, fmt.Errorf("bigquery get table %s.%s.%s: %w", projectID, datasetID, tableID, err)
	}

	// Build schema fields list.
	var fields []map[string]interface{}
	for _, f := range table.Schema.Fields {
		field := map[string]interface{}{
			"name": f.Name,
			"type": f.Type,
			"mode": f.Mode,
		}
		if f.Description != "" {
			field["description"] = f.Description
		}
		fields = append(fields, field)
	}

	result := map[string]interface{}{
		"table_id":      table.TableReference.TableId,
		"dataset_id":    table.TableReference.DatasetId,
		"project":       table.TableReference.ProjectId,
		"type":          table.Type,
		"num_rows":      table.NumRows,
		"num_bytes":     table.NumBytes,
		"creation_time": table.CreationTime,
		"last_modified": table.LastModifiedTime,
		"schema":        fields,
		"field_count":   len(fields),
	}

	if table.FriendlyName != "" {
		result["friendly_name"] = table.FriendlyName
	}
	if table.Description != "" {
		result["description"] = table.Description
	}
	if table.TimePartitioning != nil {
		result["partitioning"] = map[string]interface{}{
			"type":  table.TimePartitioning.Type,
			"field": table.TimePartitioning.Field,
		}
	}
	if table.Clustering != nil {
		result["clustering_fields"] = table.Clustering.Fields
	}
	if table.Labels != nil {
		result["labels"] = table.Labels
	}

	return result, nil
}
