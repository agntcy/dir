// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package auth provides authentication and session management logic for the Agent Hub CLI and related applications.
package auth

import (
	"context"
	"fmt"
	"time"

	hubBrowser "github.com/agntcy/dir/hub/auth/internal/browser"
	"github.com/agntcy/dir/hub/auth/internal/webserver"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/token"
)

const timeout = 60 * time.Second

// Login performs the OAuth login flow for the Agent Hub CLI.
// It starts a local webserver to handle the OAuth redirect, opens the browser for user authentication,
// exchanges the authorization code for tokens, and updates the provided session with the authenticated user and tokens.
// Returns the updated session or an error if the login process fails.
func Login(
	ctx context.Context,
	oktaClient okta.Client,
	currentSession *sessionstore.HubSession,
) (*sessionstore.HubSession, error) {
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

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	server, err := webserver.StartLocalServer(ctx, handler, config.LocalWebserverPort, errCh)
	if err != nil {
		var errChanError error
		if len(errCh) > 0 {
			errChanError = <-errCh
		}

		return nil, fmt.Errorf("failed to start local webserver: %w. error from webserver: %w", err, errChanError)
	}

	defer server.Shutdown(ctx) //nolint:errcheck

	// Open the browser
	if err := hubBrowser.OpenBrowserForLogin(currentSession.AuthConfig); err != nil {
		return nil, fmt.Errorf("could not open browser for login: %w", err)
	}

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch tokens: %w", err)
	}

	// Get tenant
	tName, err := token.GetTenantNameFromToken(webserverSession.Tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get org id: %w", err)
	}

	// Get username from token
	user, err := token.GetUserFromToken(webserverSession.Tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from token: %w", err)
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

	return currentSession, nil
}

func HasLoginCreds(currentSession *sessionstore.HubSession) bool {
	if currentSession == nil || currentSession.AuthConfig == nil {
		return false
	}

	if currentSession.CurrentTenant == "" || len(currentSession.Tokens) == 0 {
		return false
	}

	tokens, ok := currentSession.Tokens[currentSession.CurrentTenant]
	if !ok || tokens == nil {
		return false
	}

	return tokens.AccessToken != "" && tokens.IDToken != "" && tokens.RefreshToken != ""
}
