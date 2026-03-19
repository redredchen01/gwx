package api

import (
	"context"
	"fmt"
	"io"
	"os"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/slides/v1"
)

// SlidesService wraps Google Slides API operations.
type SlidesService struct {
	client *Client
}

// NewSlidesService creates a Slides service wrapper.
func NewSlidesService(client *Client) *SlidesService {
	return &SlidesService{client: client}
}

// PresentationSummary is a simplified presentation representation.
type PresentationSummary struct {
	PresentationID string       `json:"presentation_id"`
	Title          string       `json:"title"`
	SlideCount     int          `json:"slide_count"`
	Locale         string       `json:"locale,omitempty"`
	Slides         []SlideSummary `json:"slides,omitempty"`
}

// SlideSummary represents a single slide.
type SlideSummary struct {
	ObjectID   string `json:"object_id"`
	Index      int    `json:"index"`
	SlideType  string `json:"slide_type,omitempty"`
	TextCount  int    `json:"text_count"`
}

// GetPresentation retrieves a presentation's structure.
func (ss *SlidesService) GetPresentation(ctx context.Context, presentationID string) (*PresentationSummary, error) {
	opts, err := ss.client.ServiceInit(ctx, "slides")
	if err != nil {
		return nil, err
	}

	svc, err := slides.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create slides service: %w", err)
	}

	pres, err := svc.Presentations.Get(presentationID).Do()
	if err != nil {
		return nil, fmt.Errorf("get presentation: %w", err)
	}

	summary := &PresentationSummary{
		PresentationID: pres.PresentationId,
		Title:          pres.Title,
		SlideCount:     len(pres.Slides),
		Locale:         pres.Locale,
	}

	for i, s := range pres.Slides {
		ss := SlideSummary{
			ObjectID: s.ObjectId,
			Index:    i,
		}
		// Count text elements
		for _, el := range s.PageElements {
			if el.Shape != nil && el.Shape.Text != nil {
				ss.TextCount++
			}
		}
		summary.Slides = append(summary.Slides, ss)
	}

	return summary, nil
}

// ListPresentations lists presentations in Drive.
func (ss *SlidesService) ListPresentations(ctx context.Context, limit int64) ([]FileSummary, error) {
	opts, err := ss.client.ServiceInit(ctx, "drive")
	if err != nil {
		return nil, err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}

	resp, err := svc.Files.List().
		Q("mimeType='application/vnd.google-apps.presentation' and trashed=false").
		PageSize(limit).
		Fields("files(id,name,mimeType,modifiedTime,owners,webViewLink)").
		OrderBy("modifiedTime desc").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list presentations: %w", err)
	}

	var files []FileSummary
	for _, f := range resp.Files {
		files = append(files, FileSummary{
			ID:           f.Id,
			Name:         f.Name,
			MimeType:     f.MimeType,
			ModifiedTime: f.ModifiedTime,
			WebViewLink:  f.WebViewLink,
		})
	}
	return files, nil
}

// CreatePresentation creates a new empty presentation.
func (ss *SlidesService) CreatePresentation(ctx context.Context, title string) (*PresentationSummary, error) {
	opts, err := ss.client.ServiceInit(ctx, "slides")
	if err != nil {
		return nil, err
	}

	svc, err := slides.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create slides service: %w", err)
	}

	pres, err := svc.Presentations.Create(&slides.Presentation{
		Title: title,
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("create presentation: %w", err)
	}

	return &PresentationSummary{
		PresentationID: pres.PresentationId,
		Title:          pres.Title,
		SlideCount:     len(pres.Slides),
	}, nil
}

// DuplicatePresentation copies a presentation via Drive API.
func (ss *SlidesService) DuplicatePresentation(ctx context.Context, presentationID, newTitle string) (*FileSummary, error) {
	opts, err := ss.client.ServiceInit(ctx, "drive")
	if err != nil {
		return nil, err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	copied, err := svc.Files.Copy(presentationID, &drive.File{
		Name: newTitle,
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("duplicate presentation: %w", err)
	}

	return &FileSummary{
		ID:       copied.Id,
		Name:     copied.Name,
		MimeType: copied.MimeType,
	}, nil
}

// ExportPresentation exports a presentation to PDF or PPTX.
func (ss *SlidesService) ExportPresentation(ctx context.Context, presentationID, format, outputPath string) (string, error) {
	opts, err := ss.client.ServiceInit(ctx, "drive")
	if err != nil {
		return "", err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("create drive service: %w", err)
	}

	var mimeType string
	switch format {
	case "pdf":
		mimeType = "application/pdf"
	case "pptx":
		mimeType = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	default:
		return "", fmt.Errorf("unsupported format %q: use pdf or pptx", format)
	}

	// Get file name for default output path
	if outputPath == "" {
		meta, err := svc.Files.Get(presentationID).Fields("name").Do()
		if err != nil {
			return "", fmt.Errorf("get file name: %w", err)
		}
		outputPath = meta.Name + "." + format
	}

	resp, err := svc.Files.Export(presentationID, mimeType).Download()
	if err != nil {
		return "", fmt.Errorf("export presentation: %w", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return outputPath, nil
}

// FromSheet creates a presentation from a template, replacing {{placeholders}} with Sheet data.
func (ss *SlidesService) FromSheet(ctx context.Context, templateID, sheetID, sheetRange string) (*PresentationSummary, error) {
	// Step 1: Read sheet data
	sheetsSvc := NewSheetsService(ss.client)
	data, err := sheetsSvc.ReadRange(ctx, sheetID, sheetRange)
	if err != nil {
		return nil, fmt.Errorf("read sheet: %w", err)
	}
	if data == nil || len(data.Values) < 2 {
		return nil, fmt.Errorf("sheet must have at least a header row and one data row")
	}

	// Step 2: Duplicate template
	dup, err := ss.DuplicatePresentation(ctx, templateID, "Generated from Sheet")
	if err != nil {
		return nil, fmt.Errorf("duplicate template: %w", err)
	}

	// Step 3: Replace placeholders using header row as keys
	headers := data.Values[0]
	firstRow := data.Values[1] // Use first data row for replacement

	opts, err := ss.client.ServiceInit(ctx, "slides")
	if err != nil {
		return nil, err
	}

	svc, err := slides.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create slides service: %w", err)
	}

	var requests []*slides.Request
	for i, header := range headers {
		key := fmt.Sprintf("{{%v}}", header)
		value := ""
		if i < len(firstRow) {
			value = fmt.Sprintf("%v", firstRow[i])
		}
		requests = append(requests, &slides.Request{
			ReplaceAllText: &slides.ReplaceAllTextRequest{
				ContainsText: &slides.SubstringMatchCriteria{
					Text:      key,
					MatchCase: false,
				},
				ReplaceText: value,
			},
		})
	}

	if len(requests) > 0 {
		if _, err := svc.Presentations.BatchUpdate(dup.ID, &slides.BatchUpdatePresentationRequest{
			Requests: requests,
		}).Do(); err != nil {
			return nil, fmt.Errorf("replace placeholders: %w", err)
		}
	}

	// Return the new presentation info
	return ss.GetPresentation(ctx, dup.ID)
}
