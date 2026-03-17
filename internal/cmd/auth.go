package cmd

import (
	"fmt"

	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/exitcode"
	"golang.org/x/oauth2"
)

// AuthCmd manages authentication.
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Sign in to Google account"`
	Logout AuthLogoutCmd `cmd:"" help:"Remove saved credentials"`
	Status AuthStatusCmd `cmd:"" help:"Check authentication status"`
}

// AuthLoginCmd performs OAuth2 login.
type AuthLoginCmd struct {
	CredentialsFile string   `help:"Path to OAuth credentials JSON" name:"credentials" short:"c"`
	Manual          bool     `help:"Use manual (headless) auth flow" name:"manual"`
	Services        []string `help:"Services to authorize" default:"gmail,calendar,drive"`
}

func (c *AuthLoginCmd) Run(rctx *RunContext) error {
	scopes := auth.AllScopes(c.Services, false)

	if c.CredentialsFile != "" {
		if err := rctx.Auth.LoadConfigFromFile(c.CredentialsFile, scopes); err != nil {
			return fmt.Errorf("load credentials: %w", err)
		}
	} else {
		if err := rctx.Auth.LoadConfigFromKeyring(scopes); err != nil {
			return fmt.Errorf("no credentials found. Provide --credentials or run 'gwx onboard'")
		}
	}

	var token *oauth2.Token
	var err error

	if c.Manual {
		token, err = rctx.Auth.LoginManual(rctx.Context)
	} else {
		token, err = rctx.Auth.LoginBrowser(rctx.Context)
	}
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err := rctx.Auth.SaveToken(rctx.Account, token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"account": rctx.Account,
		"status":  "authenticated",
		"scopes":  scopes,
	})
	return nil
}

// AuthLogoutCmd removes saved token.
type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(rctx *RunContext) error {
	if err := rctx.Auth.DeleteToken(rctx.Account); err != nil {
		return rctx.Printer.ErrExit(exitcode.NotFound, "no saved token for account: "+rctx.Account)
	}
	rctx.Printer.Success(map[string]string{
		"account": rctx.Account,
		"status":  "logged_out",
	})
	return nil
}

// AuthStatusCmd checks auth status.
type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rctx *RunContext) error {
	if rctx.Auth.HasToken(rctx.Account) {
		rctx.Printer.Success(map[string]string{
			"account": rctx.Account,
			"status":  "authenticated",
		})
		return nil
	}
	return rctx.Printer.ErrExit(exitcode.AuthRequired, "not authenticated. Run 'gwx onboard' or 'gwx auth login'")
}
