// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package auth provides authentication and session management logic for the Agent Hub CLI and related applications.
package auth

import (
	"fmt"
	"net/http"

	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
)

// Logout logs the user out of the Agent Hub by revoking the current session's tokens.
// It uses the provided Okta client to perform the logout operation and cleans up the session data.
// Returns an error if the logout or session removal fails.
func Logout(
	opts *options.HubOptions,
	currentSession *sessionstore.HubSession,
	sessionStore sessionstore.SessionStore,
	oktaClient okta.Client,
) error {
	if err := doLogout(currentSession, oktaClient); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	// Do not remove the full session because we need to keep apikey if any. Just remove access tokens.
	currentSession.Tokens = &sessionstore.Tokens{}
	if err := sessionStore.SaveHubSession(opts.ServerAddress, currentSession); err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	return nil
}

// doLogout revokes the current session's tokens using the Okta client.
// Returns an error if the logout request fails or the response status is not OK.
func doLogout(session *sessionstore.HubSession, oktaClient okta.Client) error {
	if session == nil || session.Tokens == nil {
		return nil
	}

	resp, err := oktaClient.Logout(&okta.LogoutRequest{IDToken: session.Tokens.IDToken})
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to logout: unexpected status code: %d: %s", resp.Response.StatusCode, resp.Body)
	}

	return nil
}
