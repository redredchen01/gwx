package auth

import "fmt"

// SaveProviderToken stores a token for a non-Google provider.
// Key format: "provider:<provider>:<account>" (same as before, implemented by each backend).
// Requires NewManager() or NewManagerWithStore() to have been called first.
func SaveProviderToken(provider, account, token string) error {
	store := getDefaultStore()
	if store == nil {
		return fmt.Errorf("auth not initialized: call NewManager() first")
	}
	return store.SaveProviderToken(provider, account, token)
}

// LoadProviderToken retrieves a provider token.
// Requires NewManager() or NewManagerWithStore() to have been called first.
func LoadProviderToken(provider, account string) (string, error) {
	store := getDefaultStore()
	if store == nil {
		return "", fmt.Errorf("auth not initialized: call NewManager() first")
	}
	return store.LoadProviderToken(provider, account)
}

// DeleteProviderToken removes a provider token.
// Requires NewManager() or NewManagerWithStore() to have been called first.
func DeleteProviderToken(provider, account string) error {
	store := getDefaultStore()
	if store == nil {
		return fmt.Errorf("auth not initialized: call NewManager() first")
	}
	return store.DeleteProviderToken(provider, account)
}

// HasProviderToken checks if a token exists for the given provider and account.
// Returns false if the auth system has not been initialized.
func HasProviderToken(provider, account string) bool {
	store := getDefaultStore()
	if store == nil {
		return false
	}
	return store.HasProviderToken(provider, account)
}
