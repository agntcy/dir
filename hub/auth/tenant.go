// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package auth provides authentication and session management logic for the Agent Hub CLI and related applications.
package auth

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/agntcy/dir/hub/auth/internal/browser"
	"github.com/agntcy/dir/hub/auth/internal/webserver"
	"github.com/agntcy/dir/hub/client/idp"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/agntcy/dir/hub/utils/token"
	"github.com/manifoldco/promptui"
)

const switchTimeout = 60 * time.Second

// selectTenant prompts the user to select a tenant or uses the provided org option.
func selectTenant(tenantsMap map[string]string, opts *options.TenantSwitchOptions) (string, error) {
	if opts.Org != "" {
		return opts.Org, nil
	}

	s := promptui.Select{
		Label: "Organizations",
		Items: slices.Collect(maps.Keys(tenantsMap)),
	}

	_, selectedTenant, err := s.Run()
	if err != nil {
		return "", fmt.Errorf("interactive selection error: %w", err)
	}

	return selectedTenant, nil
}

// canReuseToken checks if a valid, non-expired token exists for the selected tenant.
func canReuseToken(currentSession *sessionstore.HubSession, selectedTenant string) bool {
	tokenData, ok := currentSession.Tokens[selectedTenant]
	if !ok {
		return false
	}

	return !token.IsTokenExpired(tokenData.AccessToken)
}

// performOAuthSwitch runs the OAuth flow for switching tenants, including starting a local webserver and opening the browser.
func performOAuthSwitch(
	ctx context.Context,
	currentSession *sessionstore.HubSession,
	oktaClient okta.Client,
	selectedTenantID string,
) (*webserver.SessionStore, error) {
	ctx, cancel := context.WithTimeout(ctx, switchTimeout)
	defer cancel()

	webserverSession := &webserver.SessionStore{}
	errChan := make(chan error, 1)
	h := webserver.NewHandler(&webserver.Config{
		ClientID:           currentSession.ClientID,
		IdpFrontendURL:     currentSession.IdpFrontendAddress,
		IdpBackendURL:      currentSession.IdpBackendAddress,
		LocalWebserverPort: config.LocalWebserverPort,
		SessionStore:       webserverSession,
		OktaClient:         oktaClient,
		ErrChan:            errChan,
	})

	server, err := webserver.StartLocalServer(ctx, h, config.LocalWebserverPort, errChan)
	if err != nil {
		var errChanErr error
		if len(errChan) > 0 {
			errChanErr = <-errChan
		}

		if server != nil {
			server.Shutdown(ctx) //nolint:errcheck
		}

		return nil, fmt.Errorf("could not start local webserver: %w. error from webserver: %w", err, errChanErr)
	}
	defer server.Shutdown(ctx) //nolint:errcheck

	if err = browser.OpenBrowserForSwitch(currentSession.AuthConfig, selectedTenantID); err != nil {
		return nil, fmt.Errorf("could not open browser: %w", err)
	}

	select {
	case err = <-errChan:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	return webserverSession, nil
}

// updateSessionWithNewTokens updates the session with new tokens for the selected tenant.
func updateSessionWithNewTokens(currentSession *sessionstore.HubSession, selectedTenant string, tokens *okta.Token) {
	currentSession.CurrentTenant = selectedTenant
	currentSession.Tokens[selectedTenant] = &sessionstore.Tokens{
		IDToken:      tokens.IDToken,
		RefreshToken: tokens.RefreshToken,
		AccessToken:  tokens.AccessToken,
	}
}

// SwitchTenant performs the tenant switch flow for the Agent Hub CLI.
// It prompts the user to select a tenant (if not provided), checks for reusable tokens,
// runs the OAuth flow if needed, and updates the session with new tokens.
// Returns the updated session, a status message, or an error if the switch fails.
func SwitchTenant(
	ctx context.Context,
	opts *options.TenantSwitchOptions,
	tenants []*idp.TenantResponse,
	currentSession *sessionstore.HubSession,
	oktaClient okta.Client,
) (*sessionstore.HubSession, string, error) {
	// Map tenants
	tenantsMap := tenantsToMap(tenants)

	// Select tenant
	selectedTenant, err := selectTenant(tenantsMap, opts)
	if err != nil {
		return nil, "", err
	}

	if selectedTenant == currentSession.CurrentTenant {
		return currentSession, "Already on tenant: " + selectedTenant, nil
	}

	if canReuseToken(currentSession, selectedTenant) {
		currentSession.CurrentTenant = selectedTenant

		return currentSession, "Switched to tenant: " + selectedTenant, nil
	}

	selectedTenantID := tenantsMap[selectedTenant]

	webserverSession, err := performOAuthSwitch(ctx, currentSession, oktaClient, selectedTenantID)
	if err != nil {
		return nil, "", err
	}

	newTenant, err := token.GetTenantNameFromToken(webserverSession.Tokens.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get org name from token: %w", err)
	}

	if newTenant != selectedTenant {
		return nil, "", fmt.Errorf("org name from token (%s) does not match selected org (%s). it could happen because you logged in another account then the one that has the requested org", newTenant, selectedTenant)
	}

	updateSessionWithNewTokens(currentSession, selectedTenant, webserverSession.Tokens)

	return currentSession, "Successfully switched to " + selectedTenant, nil //nolint:wrapcheck
}

// tenantsToMap converts a slice of TenantResponse to a map of name to ID.
func tenantsToMap(tenants []*idp.TenantResponse) map[string]string {
	m := make(map[string]string, len(tenants))
	for _, tenant := range tenants {
		m[tenant.Name] = tenant.ID
	}

	return m
}

// FetchUserTenants retrieves the list of tenants for the current user session from the IDP.
// Returns a slice of TenantResponse or an error if the request fails.
func FetchUserTenants(ctx context.Context, currentSession *sessionstore.HubSession) ([]*idp.TenantResponse, error) {
	idpClient := idp.NewClient(currentSession.AuthConfig.IdpBackendAddress, httpUtils.CreateSecureHTTPClient())
	accessToken := currentSession.Tokens[currentSession.CurrentTenant].AccessToken
	productID := currentSession.AuthConfig.IdpProductID

	idpResp, err := idpClient.GetTenantsInProduct(ctx, productID, idp.WithBearerToken(accessToken))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user tenants: %w", err)
	}

	if idpResp.TenantList == nil {
		return nil, errors.New("no tenants found")
	}

	return idpResp.TenantList.Tenants, nil
}
