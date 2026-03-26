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

// Compile-time assertion: KeyringStore must implement TokenStore.
var _ TokenStore = (*KeyringStore)(nil)

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
		return nil, fmt.Errorf("%w: %s", ErrTokenNotFound, account)
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
		return nil, fmt.Errorf("%w: credentials %s", ErrTokenNotFound, name)
	}
	var creds OAuthCredentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return &creds, nil
}

// SaveProviderToken stores a raw provider token (e.g. GitHub PAT, Slack token)
// in the OS keyring. Key format: "provider:<provider>:<account>".
func (ks *KeyringStore) SaveProviderToken(provider, account, token string) error {
	key := "provider:" + provider + ":" + account
	return keyring.Set(keyringService, key, token)
}

// LoadProviderToken retrieves a raw provider token from the OS keyring.
// Returns ErrTokenNotFound if the token does not exist.
func (ks *KeyringStore) LoadProviderToken(provider, account string) (string, error) {
	key := "provider:" + provider + ":" + account
	token, err := keyring.Get(keyringService, key)
	if err != nil {
		return "", fmt.Errorf("%w: %s/%s", ErrTokenNotFound, provider, account)
	}
	return token, nil
}

// DeleteProviderToken removes a provider token from the OS keyring.
func (ks *KeyringStore) DeleteProviderToken(provider, account string) error {
	key := "provider:" + provider + ":" + account
	return keyring.Delete(keyringService, key)
}

// HasProviderToken reports whether a token exists for the given provider and account.
func (ks *KeyringStore) HasProviderToken(provider, account string) bool {
	_, err := ks.LoadProviderToken(provider, account)
	return err == nil
}

// OAuthCredentials holds OAuth2 client ID and secret.
type OAuthCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	ProjectID    string `json:"project_id,omitempty"`
}
