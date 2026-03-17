package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// AuthMode controls how authentication is performed.
type AuthMode int

const (
	AuthBrowser  AuthMode = iota // Open browser for OAuth
	AuthManual                   // Print URL, user pastes redirect
	AuthToken                    // Direct access token (CI/CD)
	AuthADC                      // Application Default Credentials
)

// Manager handles OAuth2 authentication.
type Manager struct {
	store  *KeyringStore
	config *oauth2.Config
}

// NewManager creates an auth manager.
func NewManager() *Manager {
	return &Manager{
		store: &KeyringStore{},
	}
}

// LoadConfigFromFile reads a Google OAuth credentials JSON file
// (downloaded from Cloud Console) and configures the OAuth2 client.
func (m *Manager) LoadConfigFromFile(path string, scopes []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read credentials file: %w", err)
	}
	return m.LoadConfigFromJSON(data, scopes)
}

// LoadConfigFromJSON parses Google OAuth credentials JSON.
func (m *Manager) LoadConfigFromJSON(data []byte, scopes []string) error {
	// Google credentials JSON has a wrapper: {"installed": {...}} or {"web": {...}}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}

	var innerJSON json.RawMessage
	if v, ok := wrapper["installed"]; ok {
		innerJSON = v
	} else if v, ok := wrapper["web"]; ok {
		innerJSON = v
	} else {
		return fmt.Errorf("credentials must contain 'installed' or 'web' key")
	}

	var creds OAuthCredentials
	if err := json.Unmarshal(innerJSON, &creds); err != nil {
		return fmt.Errorf("parse inner credentials: %w", err)
	}

	// Save credentials to keyring
	if err := m.store.SaveCredentials("default", &creds); err != nil {
		return fmt.Errorf("save credentials to keyring: %w", err)
	}

	m.config = &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8085/callback",
	}
	return nil
}

// LoadConfigFromKeyring loads previously saved credentials.
func (m *Manager) LoadConfigFromKeyring(scopes []string) error {
	creds, err := m.store.LoadCredentials("default")
	if err != nil {
		return err
	}
	m.config = &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8085/callback",
	}
	return nil
}

// LoginBrowser performs OAuth2 authorization code flow with a local HTTP server.
func (m *Manager) LoginBrowser(ctx context.Context) (*oauth2.Token, error) {
	if m.config == nil {
		return nil, fmt.Errorf("OAuth config not loaded; run 'gwx onboard' first")
	}

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)
	authURL := m.config.AuthCodeURL(state, oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errCh <- fmt.Errorf("OAuth error: %s", errMsg)
			fmt.Fprintf(w, "Authorization failed: %s. You can close this tab.", errMsg)
			return
		}
		code := r.URL.Query().Get("code")
		codeCh <- code
		fmt.Fprint(w, "✓ Authorization successful! You can close this tab.")
	})

	server := &http.Server{Addr: ":8085", Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Shutdown(ctx)

	fmt.Fprintf(os.Stderr, "\nOpen this URL in your browser:\n\n  %s\n\nWaiting for authorization...\n", authURL)

	// Try to open browser
	openBrowser(authURL)

	select {
	case code := <-codeCh:
		token, err := m.config.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("exchange code: %w", err)
		}
		return token, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// LoginManual performs OAuth2 flow using a local loopback server on a random port.
// The user manually copies the URL to a browser (useful for headless/SSH environments).
func (m *Manager) LoginManual(ctx context.Context) (*oauth2.Token, error) {
	if m.config == nil {
		return nil, fmt.Errorf("OAuth config not loaded; run 'gwx onboard' first")
	}

	// Bind a random port for the callback
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for callback: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	cfg := *m.config
	cfg.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		listener.Close()
		return nil, fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errCh <- fmt.Errorf("OAuth error: %s", errMsg)
			fmt.Fprintf(w, "Authorization failed: %s", errMsg)
			return
		}
		codeCh <- r.URL.Query().Get("code")
		fmt.Fprint(w, "✓ Authorization successful! You can close this tab.")
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Shutdown(ctx)

	fmt.Fprintf(os.Stderr, "\nOpen this URL in your browser (copy-paste):\n\n  %s\n\nWaiting for authorization on 127.0.0.1:%d...\n", authURL, port)

	select {
	case code := <-codeCh:
		token, err := cfg.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("exchange code: %w", err)
		}
		return token, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TokenFromDirect creates a token source from a direct access token string.
func TokenFromDirect(accessToken string) oauth2.TokenSource {
	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	})
}

// TokenSource returns a reusable token source for the given account.
// It loads the saved token and wraps it with auto-refresh.
func (m *Manager) TokenSource(ctx context.Context, account string) (oauth2.TokenSource, error) {
	token, err := m.store.LoadToken(account)
	if err != nil {
		return nil, err
	}
	if m.config == nil {
		if err := m.LoadConfigFromKeyring(nil); err != nil {
			return nil, fmt.Errorf("load OAuth config: %w", err)
		}
	}
	return m.config.TokenSource(ctx, token), nil
}

// SaveToken saves a token to the keyring.
func (m *Manager) SaveToken(account string, token *oauth2.Token) error {
	return m.store.SaveToken(account, token)
}

// DeleteToken removes a token from the keyring.
func (m *Manager) DeleteToken(account string) error {
	return m.store.DeleteToken(account)
}

// HasToken checks if a token exists for the account.
func (m *Manager) HasToken(account string) bool {
	_, err := m.store.LoadToken(account)
	return err == nil
}

// openBrowser tries to open a URL in the default browser.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return
	}
	_ = commandExec(cmd, args...).Run()
}

