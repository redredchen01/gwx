package cmd

import (
	"sync"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// UnifiedSearchCmd searches across Gmail, Drive, and Contacts simultaneously.
type UnifiedSearchCmd struct {
	Query    string   `arg:"" help:"Search query"`
	Services []string `help:"Services to search (gmail,drive,contacts)" default:"gmail,drive" short:"s"`
	Limit    int      `help:"Max results per service" default:"5" short:"n"`
}

// SearchResultGroup holds results from one service.
type SearchResultGroup struct {
	Service string      `json:"service"`
	Count   int         `json:"count"`
	Results interface{} `json:"results"`
	Error   string      `json:"error,omitempty"`
}

func (c *UnifiedSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "unified.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	// Collect needed services for auth
	authServices := make([]string, 0, len(c.Services))
	for _, s := range c.Services {
		switch s {
		case "gmail":
			authServices = append(authServices, "gmail")
		case "drive":
			authServices = append(authServices, "drive")
		case "contacts":
			authServices = append(authServices, "people")
		}
	}

	if err := EnsureAuth(rctx, authServices); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":  "unified.search",
			"query":    c.Query,
			"services": c.Services,
		})
		return nil
	}

	// Parallel search across services
	var mu sync.Mutex
	var groups []SearchResultGroup
	var wg sync.WaitGroup

	serviceSet := make(map[string]bool)
	for _, s := range c.Services {
		serviceSet[s] = true
	}

	if serviceSet["gmail"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := api.NewGmailService(rctx.APIClient)
			messages, total, err := svc.SearchMessages(rctx.Context, c.Query, int64(c.Limit))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				groups = append(groups, SearchResultGroup{Service: "gmail", Error: err.Error()})
				return
			}
			groups = append(groups, SearchResultGroup{
				Service: "gmail",
				Count:   len(messages),
				Results: map[string]interface{}{
					"messages":       messages,
					"total_estimate": total,
				},
			})
		}()
	}

	if serviceSet["drive"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := api.NewDriveService(rctx.APIClient)
			// Use fullText search for Drive
			driveQuery := "fullText contains '" + escapeDriveQuery(c.Query) + "'"
			files, err := svc.SearchFiles(rctx.Context, driveQuery, int64(c.Limit))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				// Fallback to name search
				files, err = svc.SearchFiles(rctx.Context, "name contains '"+escapeDriveQuery(c.Query)+"'", int64(c.Limit))
				if err != nil {
					groups = append(groups, SearchResultGroup{Service: "drive", Error: err.Error()})
					return
				}
			}
			groups = append(groups, SearchResultGroup{
				Service: "drive",
				Count:   len(files),
				Results: files,
			})
		}()
	}

	if serviceSet["contacts"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := api.NewContactsService(rctx.APIClient)
			contacts, err := svc.SearchContacts(rctx.Context, c.Query, c.Limit)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				groups = append(groups, SearchResultGroup{Service: "contacts", Error: err.Error()})
				return
			}
			groups = append(groups, SearchResultGroup{
				Service: "contacts",
				Count:   len(contacts),
				Results: contacts,
			})
		}()
	}

	wg.Wait()

	totalResults := 0
	for _, g := range groups {
		totalResults += g.Count
	}

	rctx.Printer.Success(map[string]interface{}{
		"query":         c.Query,
		"total_results": totalResults,
		"results":       groups,
	})
	return nil
}

func escapeDriveQuery(s string) string {
	// Escape single quotes for Drive query
	var result []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			result = append(result, '\\', '\'')
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}
