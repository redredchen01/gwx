package auth

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const providerKeyPrefix = "provider:"

// SaveProviderToken stores a token for a non-Google provider in the OS keyring.
// The key format is "provider:<provider>:<account>" to avoid collisions
// with Google OAuth tokens which use "token:<account>".
func SaveProviderToken(provider, account, token string) error {
	key := providerKeyPrefix + provider + ":" + account
	return keyring.Set(keyringService, key, token)
}

// LoadProviderToken retrieves a provider token from the OS keyring.
func LoadProviderToken(provider, account string) (string, error) {
	key := providerKeyPrefix + provider + ":" + account
	token, err := keyring.Get(keyringService, key)
	if err != nil {
		return "", fmt.Errorf("no %s token found for account %q: %w", provider, account, err)
	}
	return token, nil
}

// DeleteProviderToken removes a provider token from the OS keyring.
func DeleteProviderToken(provider, account string) error {
	key := providerKeyPrefix + provider + ":" + account
	return keyring.Delete(keyringService, key)
}

// HasProviderToken checks if a token exists for the given provider and account.
func HasProviderToken(provider, account string) bool {
	_, err := LoadProviderToken(provider, account)
	return err == nil
}
