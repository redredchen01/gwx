package api

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/people/v1"
)

// ContactsService wraps People API operations.
type ContactsService struct {
	client *Client
}

// NewContactsService creates a Contacts service wrapper.
func NewContactsService(client *Client) *ContactsService {
	return &ContactsService{client: client}
}

// ContactSummary is a simplified contact.
type ContactSummary struct {
	ResourceName string   `json:"resource_name"`
	Name         string   `json:"name"`
	Emails       []string `json:"emails,omitempty"`
	Phones       []string `json:"phones,omitempty"`
	Organization string   `json:"organization,omitempty"`
	Title        string   `json:"title,omitempty"`
	Photo        string   `json:"photo,omitempty"`
}

const personFields = "names,emailAddresses,phoneNumbers,organizations,photos"

// ListContacts lists contacts.
func (cs *ContactsService) ListContacts(ctx context.Context, maxResults int) ([]ContactSummary, error) {
	if err := cs.client.WaitRate(ctx, "people"); err != nil {
		return nil, err
	}

	opts, err := cs.client.ClientOptions(ctx, "people")
	if err != nil {
		return nil, err
	}

	svc, err := people.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create people service: %w", err)
	}

	if maxResults <= 0 {
		maxResults = 100
	}

	call := svc.People.Connections.List("people/me").
		PersonFields(personFields).
		PageSize(int64(maxResults)).
		SortOrder("LAST_MODIFIED_DESCENDING")

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}

	var contacts []ContactSummary
	for _, p := range resp.Connections {
		contacts = append(contacts, personToContact(p))
	}
	return contacts, nil
}

// SearchContacts searches contacts by name or email.
func (cs *ContactsService) SearchContacts(ctx context.Context, query string, maxResults int) ([]ContactSummary, error) {
	if err := cs.client.WaitRate(ctx, "people"); err != nil {
		return nil, err
	}

	opts, err := cs.client.ClientOptions(ctx, "people")
	if err != nil {
		return nil, err
	}

	svc, err := people.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create people service: %w", err)
	}

	if maxResults <= 0 {
		maxResults = 30
	}

	resp, err := svc.People.SearchContacts().
		Query(query).
		ReadMask(personFields).
		PageSize(int64(maxResults)).
		Do()
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}

	var contacts []ContactSummary
	for _, r := range resp.Results {
		if r.Person != nil {
			contacts = append(contacts, personToContact(r.Person))
		}
	}
	return contacts, nil
}

// GetContact retrieves a specific contact.
func (cs *ContactsService) GetContact(ctx context.Context, resourceName string) (*ContactSummary, error) {
	if err := cs.client.WaitRate(ctx, "people"); err != nil {
		return nil, err
	}

	opts, err := cs.client.ClientOptions(ctx, "people")
	if err != nil {
		return nil, err
	}

	svc, err := people.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create people service: %w", err)
	}

	if !strings.HasPrefix(resourceName, "people/") {
		resourceName = "people/" + resourceName
	}

	p, err := svc.People.Get(resourceName).PersonFields(personFields).Do()
	if err != nil {
		return nil, fmt.Errorf("get contact: %w", err)
	}

	contact := personToContact(p)
	return &contact, nil
}

func personToContact(p *people.Person) ContactSummary {
	c := ContactSummary{
		ResourceName: p.ResourceName,
	}

	if len(p.Names) > 0 {
		c.Name = p.Names[0].DisplayName
	}

	for _, e := range p.EmailAddresses {
		c.Emails = append(c.Emails, e.Value)
	}

	for _, ph := range p.PhoneNumbers {
		c.Phones = append(c.Phones, ph.Value)
	}

	if len(p.Organizations) > 0 {
		c.Organization = p.Organizations[0].Name
		c.Title = p.Organizations[0].Title
	}

	if len(p.Photos) > 0 {
		c.Photo = p.Photos[0].Url
	}

	return c
}
