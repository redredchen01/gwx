package workflow

import (
	"context"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
)

// EmailFromDocResult is the output of RunEmailFromDoc.
type EmailFromDocResult struct {
	DocID      string         `json:"doc_id"`
	DocTitle   string         `json:"doc_title"`
	Recipients []string      `json:"recipients"`
	Preview    string         `json:"preview"`
	Execute    *ExecuteResult `json:"execute,omitempty"`
}

// EmailFromDocOpts configures the email-from-doc workflow.
type EmailFromDocOpts struct {
	DocID      string
	Recipients []string
	Subject    string
	Execute    bool
	NoInput    bool
	IsMCP      bool
}

// RunEmailFromDoc reads a Google Doc and optionally sends it as an email.
func RunEmailFromDoc(ctx context.Context, client *api.Client, opts EmailFromDocOpts) (*EmailFromDocResult, error) {
	docSvc := api.NewDocsService(client)
	doc, err := docSvc.GetDocument(ctx, opts.DocID)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}

	subject := opts.Subject
	if subject == "" {
		subject = doc.Title
	}

	// Truncate preview
	bodyPreview := doc.Body
	if len(bodyPreview) > 500 {
		bodyPreview = bodyPreview[:500] + "..."
	}

	result := &EmailFromDocResult{
		DocID:      doc.DocumentID,
		DocTitle:   doc.Title,
		Recipients: opts.Recipients,
		Preview:    fmt.Sprintf("Subject: %s\n\n%s", subject, bodyPreview),
	}

	if opts.Execute && len(opts.Recipients) > 0 {
		actions := []Action{
			{
				Name:        "send_email",
				Description: fmt.Sprintf("Send doc '%s' to %v", doc.Title, opts.Recipients),
				Fn: func(ctx context.Context) (interface{}, error) {
					svc := api.NewGmailService(client)
					return svc.SendMessage(ctx, &api.SendInput{
						To:      opts.Recipients,
						Subject: subject,
						Body:    doc.Body,
					})
				},
			},
		}

		execResult, err := Dispatch(ctx, actions, ExecuteOpts{
			Execute: opts.Execute,
			NoInput: opts.NoInput,
			IsMCP:   opts.IsMCP,
		})
		if err != nil {
			return nil, err
		}
		result.Execute = execResult
	}

	return result, nil
}
