package auth

import (
	"errors"

	"golang.org/x/oauth2"
)

// TokenStore is the unified credential storage interface.
// Implementations: KeyringStore (OS keyring), FileStore (encrypted file).
type TokenStore interface {
	// OAuth2 token (structured)
	SaveToken(account string, token *oauth2.Token) error
	LoadToken(account string) (*oauth2.Token, error)
	DeleteToken(account string) error

	// OAuth2 client credentials
	SaveCredentials(name string, creds *OAuthCredentials) error
	LoadCredentials(name string) (*OAuthCredentials, error)

	// Provider token (raw string — GitHub PAT, Slack token, etc.)
	SaveProviderToken(provider, account, token string) error
	LoadProviderToken(provider, account string) (string, error)
	DeleteProviderToken(provider, account string) error
	HasProviderToken(provider, account string) bool
}

// Sentinel errors for TokenStore operations.
var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrCredentialCorrupted = errors.New("credential file corrupted")
	ErrKeyMismatch        = errors.New("encryption key mismatch (wrong machine?)")
	ErrLockTimeout        = errors.New("file lock timeout")
)
