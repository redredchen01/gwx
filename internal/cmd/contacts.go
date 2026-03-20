package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
)

// ContactsCmd groups Contacts operations.
type ContactsCmd struct {
	List   ContactsListCmd   `cmd:"" help:"List contacts"`
	Search ContactsSearchCmd `cmd:"" help:"Search contacts"`
	Get    ContactsGetCmd    `cmd:"" help:"Get a contact"`
}

// ContactsListCmd lists contacts.
type ContactsListCmd struct {
	Limit int `help:"Max contacts to return" default:"50" short:"n"`
}

func (c *ContactsListCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "contacts.list", []string{"people"}); done {
		return err
	}

	contactsSvc := api.NewContactsService(rctx.APIClient)
	contacts, err := contactsSvc.ListContacts(rctx.Context, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"contacts": contacts,
		"count":    len(contacts),
	})
	return nil
}

// ContactsSearchCmd searches contacts.
type ContactsSearchCmd struct {
	Query string `arg:"" help:"Search query (name or email)"`
	Limit int    `help:"Max results" default:"20" short:"n"`
}

func (c *ContactsSearchCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "contacts.search", []string{"people"}); done {
		return err
	}

	contactsSvc := api.NewContactsService(rctx.APIClient)
	contacts, err := contactsSvc.SearchContacts(rctx.Context, c.Query, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"query":    c.Query,
		"contacts": contacts,
		"count":    len(contacts),
	})
	return nil
}

// ContactsGetCmd gets a contact.
type ContactsGetCmd struct {
	ResourceName string `arg:"" help:"Contact resource name (e.g. people/c123)"`
}

func (c *ContactsGetCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "contacts.get", []string{"people"}); done {
		return err
	}

	contactsSvc := api.NewContactsService(rctx.APIClient)
	contact, err := contactsSvc.GetContact(rctx.Context, c.ResourceName)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(contact)
	return nil
}
