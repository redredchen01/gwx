package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redredchen01/gwx/internal/exitcode"
	"golang.org/x/oauth2"
)

// TokenExport is the envelope for exported tokens.
type TokenExport struct {
	Version    int           `json:"version"`
	Account    string        `json:"account"`
	ExportedAt time.Time     `json:"exported_at"`
	Token      *oauth2.Token `json:"token"`
}

// AuthExportCmd exports the stored OAuth2 token as JSON to stdout.
type AuthExportCmd struct{}

func (c *AuthExportCmd) Run(rctx *RunContext) error {
	token, err := rctx.Auth.LoadToken(rctx.Account)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.NotFound, "no token for account: "+rctx.Account)
	}

	export := TokenExport{
		Version:    1,
		Account:    rctx.Account,
		ExportedAt: time.Now().UTC(),
		Token:      token,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(export); err != nil {
		return fmt.Errorf("encode token: %w", err)
	}

	if !token.Expiry.IsZero() && token.Expiry.Before(time.Now()) {
		fmt.Fprintln(os.Stderr, "warning: token is expired (refresh_token may still work)")
	}
	return nil
}

// AuthImportCmd imports a token from JSON.
type AuthImportCmd struct {
	JSON string `help:"Token JSON string" name:"json"`
}

func (c *AuthImportCmd) Run(rctx *RunContext) error {
	data, source, err := readJSONInput(c.JSON, isStdinPipe())
	if err != nil {
		return err
	}

	token, envAccount, err := validateTokenImport(data)
	if err != nil {
		return fmt.Errorf("invalid token (%s): %w", source, err)
	}

	account := rctx.Account
	if account == "default" && envAccount != "" && envAccount != "default" {
		account = envAccount
		fmt.Fprintf(os.Stderr, "using account from export: %s\n", account)
	}

	if err := rctx.Auth.SaveToken(account, token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	if !token.Expiry.IsZero() && token.Expiry.Before(time.Now()) {
		fmt.Fprintln(os.Stderr, "warning: imported token is expired (refresh_token may still work)")
	}

	rctx.Printer.Success(map[string]interface{}{
		"account": account,
		"status":  "imported",
		"source":  source,
	})
	return nil
}

// validateTokenImport parses token JSON, accepting either envelope or raw oauth2.Token.
func validateTokenImport(data []byte) (*oauth2.Token, string, error) {
	// Try envelope first
	var env TokenExport
	if err := json.Unmarshal(data, &env); err == nil && env.Version > 0 && env.Token != nil {
		if env.Token.AccessToken == "" && env.Token.RefreshToken == "" {
			return nil, "", fmt.Errorf("token has neither access_token nor refresh_token")
		}
		return env.Token, env.Account, nil
	}

	// Fall back to raw oauth2.Token
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, "", fmt.Errorf("invalid token JSON: %w", err)
	}
	if tok.AccessToken == "" && tok.RefreshToken == "" {
		return nil, "", fmt.Errorf("token has neither access_token nor refresh_token")
	}
	return &tok, "", nil
}
