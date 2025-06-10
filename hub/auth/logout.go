package auth

import (
	"fmt"
	"net/http"

	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
)

func Logout(
	opts *options.HubOptions,
	currentSession *sessionstore.HubSession,
	sessionStore sessionstore.SessionStore,
	oktaClient okta.Client,
) error {
	if err := doLogout(currentSession, oktaClient); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	if err := sessionStore.RemoveSession(opts.ServerAddress); err != nil {
		return fmt.Errorf("failed to remove session: %w", err)
	}

	return nil
}

func doLogout(session *sessionstore.HubSession, oktaClient okta.Client) error {
	if session == nil || session.CurrentTenant == "" || session.Tokens == nil {
		return nil
	}

	// Check if the session exists
	if _, ok := session.Tokens[session.CurrentTenant]; !ok {
		return nil
	}

	resp, err := oktaClient.Logout(&okta.LogoutRequest{IDToken: session.Tokens[session.CurrentTenant].IDToken})
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to logout: unexpected status code: %d: %s", resp.Response.StatusCode, resp.Body)
	}

	return nil
}
