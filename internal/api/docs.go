package api

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
)

// DocsService wraps Google Docs API operations.
type DocsService struct {
	client *Client
}

// NewDocsService creates a Docs service wrapper.
func NewDocsService(client *Client) *DocsService {
	return &DocsService{client: client}
}

// DocSummary is a simplified document representation.
type DocSummary struct {
	DocumentID string `json:"document_id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	RevisionID string `json:"revision_id,omitempty"`
}

// GetDocument retrieves a document and extracts plain text.
func (ds *DocsService) GetDocument(ctx context.Context, docID string) (*DocSummary, error) {
	if err := ds.client.WaitRate(ctx, "docs"); err != nil {
		return nil, err
	}

	opts, err := ds.client.ClientOptions(ctx, "docs")
	if err != nil {
		return nil, err
	}

	svc, err := docs.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create docs service: %w", err)
	}

	doc, err := svc.Documents.Get(docID).Do()
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}

	body := extractDocText(doc.Body)

	return &DocSummary{
		DocumentID: doc.DocumentId,
		Title:      doc.Title,
		Body:       body,
		RevisionID: doc.RevisionId,
	}, nil
}

// CreateDocument creates a new Google Doc.
func (ds *DocsService) CreateDocument(ctx context.Context, title string, body string) (*DocSummary, error) {
	if err := ds.client.WaitRate(ctx, "docs"); err != nil {
		return nil, err
	}

	opts, err := ds.client.ClientOptions(ctx, "docs")
	if err != nil {
		return nil, err
	}

	svc, err := docs.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create docs service: %w", err)
	}

	doc := &docs.Document{Title: title}
	created, err := svc.Documents.Create(doc).Do()
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	// If body content provided, insert it
	if body != "" {
		if err := ds.client.WaitRate(ctx, "docs"); err != nil {
			return nil, err
		}

		req := &docs.BatchUpdateDocumentRequest{
			Requests: []*docs.Request{
				{
					InsertText: &docs.InsertTextRequest{
						Text: body,
						Location: &docs.Location{
							Index: 1,
						},
					},
				},
			},
		}
		if _, err := svc.Documents.BatchUpdate(created.DocumentId, req).Do(); err != nil {
			return nil, fmt.Errorf("insert body text: %w", err)
		}
	}

	return &DocSummary{
		DocumentID: created.DocumentId,
		Title:      created.Title,
		Body:       body,
	}, nil
}

// AppendText appends text to the end of a document.
func (ds *DocsService) AppendText(ctx context.Context, docID string, text string) error {
	if err := ds.client.WaitRate(ctx, "docs"); err != nil {
		return err
	}

	opts, err := ds.client.ClientOptions(ctx, "docs")
	if err != nil {
		return err
	}

	svc, err := docs.NewService(ctx, opts...)
	if err != nil {
		return fmt.Errorf("create docs service: %w", err)
	}

	// Get document to find end index
	doc, err := svc.Documents.Get(docID).Do()
	if err != nil {
		return fmt.Errorf("get document for append: %w", err)
	}

	endIndex := doc.Body.Content[len(doc.Body.Content)-1].EndIndex - 1

	req := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Text: "\n" + text,
					Location: &docs.Location{
						Index: endIndex,
					},
				},
			},
		},
	}

	if _, err := svc.Documents.BatchUpdate(docID, req).Do(); err != nil {
		return fmt.Errorf("append text: %w", err)
	}
	return nil
}

// ExportDocument exports a document to a file (PDF, DOCX, etc.).
func (ds *DocsService) ExportDocument(ctx context.Context, docID string, format string, outputPath string) (string, error) {
	if err := ds.client.WaitRate(ctx, "drive"); err != nil {
		return "", err
	}

	opts, err := ds.client.ClientOptions(ctx, "drive")
	if err != nil {
		return "", err
	}

	drvSvc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("create drive service: %w", err)
	}

	mimeType := exportMimeType(format)
	if mimeType == "" {
		return "", fmt.Errorf("unsupported export format: %s (use pdf, docx, txt, html)", format)
	}

	resp, err := drvSvc.Files.Export(docID, mimeType).Download()
	if err != nil {
		return "", fmt.Errorf("export document: %w", err)
	}
	defer resp.Body.Close()

	if outputPath == "" {
		outputPath = docID + "." + format
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("write export: %w", err)
	}

	return outputPath, nil
}

func extractDocText(body *docs.Body) string {
	if body == nil {
		return ""
	}
	var sb strings.Builder
	for _, elem := range body.Content {
		if elem.Paragraph != nil {
			for _, pe := range elem.Paragraph.Elements {
				if pe.TextRun != nil {
					sb.WriteString(pe.TextRun.Content)
				}
			}
		}
	}
	return sb.String()
}

func exportMimeType(format string) string {
	switch strings.ToLower(format) {
	case "pdf":
		return "application/pdf"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "txt":
		return "text/plain"
	case "html":
		return "text/html"
	default:
		return ""
	}
}
