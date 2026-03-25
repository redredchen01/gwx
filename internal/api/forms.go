package api

import (
	"context"
	"fmt"

	forms "google.golang.org/api/forms/v1"
)

func (fs *FormsService) service(ctx context.Context) (*forms.Service, error) {
	svc, err := fs.client.GetOrCreateService("forms:v1", func() (any, error) {
		opts, err := fs.client.ClientOptions(ctx, "forms")
		if err != nil {
			return nil, err
		}
		return forms.NewService(ctx, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("create forms service: %w", err)
	}
	return svc.(*forms.Service), nil
}

// FormsService wraps Google Forms API operations.
type FormsService struct {
	client *Client
}

// NewFormsService creates a Forms service wrapper.
func NewFormsService(client *Client) *FormsService {
	return &FormsService{client: client}
}

// GetForm retrieves the structure of a Google Form by its ID.
func (fs *FormsService) GetForm(ctx context.Context, formID string) (map[string]interface{}, error) {
	if err := fs.client.WaitRate(ctx, "forms"); err != nil {
		return nil, err
	}

	svc, err := fs.service(ctx)
	if err != nil {
		return nil, err
	}

	form, err := svc.Forms.Get(formID).Do()
	if err != nil {
		return nil, fmt.Errorf("forms get %s: %w", formID, err)
	}

	// Build items list from form questions.
	var items []map[string]interface{}
	for _, item := range form.Items {
		entry := map[string]interface{}{
			"item_id": item.ItemId,
			"title":   item.Title,
		}
		if item.Description != "" {
			entry["description"] = item.Description
		}
		if item.QuestionItem != nil && item.QuestionItem.Question != nil {
			q := item.QuestionItem.Question
			entry["question_id"] = q.QuestionId
			entry["required"] = q.Required

			switch {
			case q.ChoiceQuestion != nil:
				entry["type"] = "choice"
				entry["choice_type"] = q.ChoiceQuestion.Type
				var options []string
				for _, opt := range q.ChoiceQuestion.Options {
					options = append(options, opt.Value)
				}
				entry["options"] = options
			case q.TextQuestion != nil:
				entry["type"] = "text"
				entry["paragraph"] = q.TextQuestion.Paragraph
			case q.ScaleQuestion != nil:
				entry["type"] = "scale"
				entry["low"] = q.ScaleQuestion.Low
				entry["high"] = q.ScaleQuestion.High
				entry["low_label"] = q.ScaleQuestion.LowLabel
				entry["high_label"] = q.ScaleQuestion.HighLabel
			case q.DateQuestion != nil:
				entry["type"] = "date"
				entry["include_time"] = q.DateQuestion.IncludeTime
				entry["include_year"] = q.DateQuestion.IncludeYear
			case q.TimeQuestion != nil:
				entry["type"] = "time"
				entry["duration"] = q.TimeQuestion.Duration
			case q.FileUploadQuestion != nil:
				entry["type"] = "file_upload"
			default:
				entry["type"] = "unknown"
			}
		}
		if item.QuestionGroupItem != nil {
			entry["type"] = "question_group"
			var subQuestions []map[string]interface{}
			if item.QuestionGroupItem.Grid != nil && item.QuestionGroupItem.Grid.Columns != nil {
				var cols []string
				for _, opt := range item.QuestionGroupItem.Grid.Columns.Options {
					cols = append(cols, opt.Value)
				}
				entry["grid_columns"] = cols
			}
			for _, q := range item.QuestionGroupItem.Questions {
				sq := map[string]interface{}{
					"question_id": q.QuestionId,
					"required":    q.Required,
				}
				if q.RowQuestion != nil {
					sq["title"] = q.RowQuestion.Title
				}
				subQuestions = append(subQuestions, sq)
			}
			entry["questions"] = subQuestions
		}
		if item.PageBreakItem != nil {
			entry["type"] = "page_break"
		}
		if item.TextItem != nil {
			entry["type"] = "text_block"
		}
		if item.ImageItem != nil {
			entry["type"] = "image"
		}
		if item.VideoItem != nil {
			entry["type"] = "video"
		}
		items = append(items, entry)
	}

	result := map[string]interface{}{
		"form_id":       form.FormId,
		"title":         form.Info.Title,
		"document_title": form.Info.DocumentTitle,
		"revision_id":   form.RevisionId,
		"responder_uri": form.ResponderUri,
		"item_count":    len(items),
		"items":         items,
	}

	if form.Info.Description != "" {
		result["description"] = form.Info.Description
	}

	return result, nil
}

// ListResponses lists responses for a Google Form.
func (fs *FormsService) ListResponses(ctx context.Context, formID string, limit int) ([]map[string]interface{}, error) {
	if err := fs.client.WaitRate(ctx, "forms"); err != nil {
		return nil, err
	}

	svc, err := fs.service(ctx)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	// The Forms API paginates responses; collect up to limit.
	var all []map[string]interface{}
	pageToken := ""

	for {
		call := svc.Forms.Responses.List(formID)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		// The API uses PageSize to control batch size.
		batchSize := int64(limit - len(all))
		if batchSize > 5000 {
			batchSize = 5000
		}
		call = call.PageSize(batchSize)

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("forms list responses %s: %w", formID, err)
		}

		for _, r := range resp.Responses {
			entry := responseToMap(r)
			all = append(all, entry)
			if len(all) >= limit {
				return all, nil
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return all, nil
}

// GetResponse retrieves a single form response by ID.
func (fs *FormsService) GetResponse(ctx context.Context, formID, responseID string) (map[string]interface{}, error) {
	if err := fs.client.WaitRate(ctx, "forms"); err != nil {
		return nil, err
	}

	svc, err := fs.service(ctx)
	if err != nil {
		return nil, err
	}

	r, err := svc.Forms.Responses.Get(formID, responseID).Do()
	if err != nil {
		return nil, fmt.Errorf("forms get response %s/%s: %w", formID, responseID, err)
	}

	return responseToMap(r), nil
}

// responseToMap converts a Forms API response to a simple map.
func responseToMap(r *forms.FormResponse) map[string]interface{} {
	answers := make(map[string]interface{})
	for qID, ans := range r.Answers {
		a := map[string]interface{}{
			"question_id": ans.QuestionId,
		}
		if ans.TextAnswers != nil {
			var values []string
			for _, ta := range ans.TextAnswers.Answers {
				values = append(values, ta.Value)
			}
			a["text_answers"] = values
		}
		if ans.FileUploadAnswers != nil {
			var files []map[string]interface{}
			for _, fu := range ans.FileUploadAnswers.Answers {
				files = append(files, map[string]interface{}{
					"file_id":  fu.FileId,
					"file_name": fu.FileName,
					"mime_type": fu.MimeType,
				})
			}
			a["file_answers"] = files
		}
		answers[qID] = a
	}

	entry := map[string]interface{}{
		"response_id":       r.ResponseId,
		"create_time":       r.CreateTime,
		"last_submitted_time": r.LastSubmittedTime,
		"answers":           answers,
	}
	if r.RespondentEmail != "" {
		entry["respondent_email"] = r.RespondentEmail
	}
	if r.TotalScore != 0 {
		entry["total_score"] = r.TotalScore
	}
	return entry
}
