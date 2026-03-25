package api

import (
	"testing"

	"golang.org/x/oauth2"
)

func TestClientHTTPClientReusesPerServiceClient(t *testing.T) {
	client := NewClient(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}))

	first := client.HTTPClient("gmail")
	second := client.HTTPClient("gmail")

	if first != second {
		t.Fatal("expected HTTPClient to reuse the same client for the same service")
	}
}

func TestClientHTTPClientSeparatesServices(t *testing.T) {
	client := NewClient(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}))

	gmailClient := client.HTTPClient("gmail")
	driveClient := client.HTTPClient("drive")

	if gmailClient == driveClient {
		t.Fatal("expected distinct HTTP clients for different services")
	}
}
