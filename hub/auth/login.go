package auth

import (
	"context"
	"fmt"
	"io"
	"time"

	hubBrowser "github.com/agntcy/dir/hub/auth/internal/browser"
	"github.com/agntcy/dir/hub/auth/internal/webserver"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/token"
)

const timeout = 60 * time.Second

func Login(
	out io.Writer,
	opts *options.LoginOptions,
	oktaClient okta.Client,
	sessionStore sessionstore.SessionStore,
	currentSession *sessionstore.HubSession,
) error {
	// Set up the webserver
	errCh := make(chan error, 1)
	webserverSession := &webserver.SessionStore{}

	handler := webserver.NewHandler(&webserver.Config{
		ClientID:           currentSession.AuthConfig.ClientID,
		IdpFrontendURL:     currentSession.AuthConfig.IdpFrontendAddress,
		IdpBackendURL:      currentSession.AuthConfig.IdpBackendAddress,
		LocalWebserverPort: config.LocalWebserverPort,
		OktaClient:         oktaClient,
		SessionStore:       webserverSession,
		ErrChan:            errCh,
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	server, err := webserver.StartLocalServer(handler, config.LocalWebserverPort, errCh)
	if err != nil {
		var errChanError error
		if len(errCh) > 0 {
			errChanError = <-errCh
		}
		return fmt.Errorf("failed to start local webserver: %w. error from webserver: %w", err, errChanError)
	}
	defer server.Shutdown(ctx) //nolint:errcheck

	// Open the browser
	if err := hubBrowser.OpenBrowserForLogin(currentSession.AuthConfig); err != nil {
		return fmt.Errorf("could not open browser for login: %w", err)
	}

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		return fmt.Errorf("failed to fetch tokens: %w", err)
	}

	// Get tenant
	tName, err := token.GetTenantNameFromToken(webserverSession.Tokens.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get org id: %w", err)
	}

	// Get username from token
	user, err := token.GetUserFromToken(webserverSession.Tokens.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get user from token: %w", err)
	}

	currentSession.Tokens = make(map[string]*sessionstore.Tokens)
	// Set current tenant
	currentSession.CurrentTenant = tName
	// Set user
	currentSession.User = user
	// Set tokens
	currentSession.Tokens[tName] = &sessionstore.Tokens{
		AccessToken:  webserverSession.Tokens.AccessToken,
		RefreshToken: webserverSession.Tokens.RefreshToken,
		IDToken:      webserverSession.Tokens.IDToken,
	}

	// Get tokens
	err = sessionStore.SaveHubSession(opts.ServerAddress, currentSession)
	if err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Successfully logged in to Agent Hub\nAddress: %s\nUser: %s\nOrganization: %s\n", opts.ServerAddress, user, tName)

	return nil
}
