package auth

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	keyringService = "gwx"
	tokenKeyPrefix = "token:"
	credKeyPrefix  = "cred:"
)

// KeyringStore manages OAuth tokens in the OS keyring.
// Tokens never touch the filesystem.
type KeyringStore struct{}

// SaveToken stores an OAuth2 token in the OS keyring.
func (ks *KeyringStore) SaveToken(account string, token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	return keyring.Set(keyringService, tokenKeyPrefix+account, string(data))
}

// LoadToken retrieves an OAuth2 token from the OS keyring.
func (ks *KeyringStore) LoadToken(account string) (*oauth2.Token, error) {
	data, err := keyring.Get(keyringService, tokenKeyPrefix+account)
	if err != nil {
		return nil, fmt.Errorf("token not found for %s: %w", account, err)
	}
	var token oauth2.Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	return &token, nil
}

// DeleteToken removes an OAuth2 token from the OS keyring.
func (ks *KeyringStore) DeleteToken(account string) error {
	return keyring.Delete(keyringService, tokenKeyPrefix+account)
}

// SaveCredentials stores OAuth client credentials in the OS keyring.
func (ks *KeyringStore) SaveCredentials(name string, creds *OAuthCredentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	return keyring.Set(keyringService, credKeyPrefix+name, string(data))
}

// LoadCredentials retrieves OAuth client credentials from the OS keyring.
func (ks *KeyringStore) LoadCredentials(name string) (*OAuthCredentials, error) {
	data, err := keyring.Get(keyringService, credKeyPrefix+name)
	if err != nil {
		return nil, fmt.Errorf("credentials not found for %s: %w", name, err)
	}
	var creds OAuthCredentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return &creds, nil
}

// OAuthCredentials holds OAuth2 client ID and secret.
type OAuthCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	ProjectID    string `json:"project_id,omitempty"`
}
