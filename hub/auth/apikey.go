// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/agntcy/dir/hub/auth/utils"
	"github.com/agntcy/dir/hub/client/idp"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/file"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
)

// RefreshAPIKeyAccessToken refreshes API key access token and saves to session file.
func RefreshAPIKeyAccessToken(ctx context.Context, session *sessionstore.HubSession, sessionKey string) error {
	if err := retrieveAccessTokenWithAPIKey(ctx, session); err != nil {
		return fmt.Errorf("failed to refresh API key access token: %w", err)
	}

	// Load session store for saving
	sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())

	// Save updated session
	if err := sessionStore.SaveHubSession(sessionKey, session); err != nil {
		return fmt.Errorf("failed to save sessions: %w", err)
	}

	return nil
}

// CreateInMemorySessionFromAPIKey authenticates via API key for the CLI without a session file.
func CreateInMemorySessionFromAPIKey(ctx context.Context, serverAddress, clientID, secret string) (*sessionstore.HubSession, error) {
	if clientID == "" || secret == "" {
		return nil, errors.New("both client ID and secret must be provided")
	}

	// Fetch auth configuration from server.
	authConfig, err := config.FetchAuthConfig(ctx, serverAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch auth config: %w", err)
	}

	// Create in-memory session.
	session := &sessionstore.HubSession{
		AuthConfig: &sessionstore.AuthConfig{
			ClientID:           authConfig.ClientID,
			IdpProductID:       authConfig.IdpProductID,
			IdpFrontendAddress: authConfig.IdpFrontendAddress,
			IdpBackendAddress:  authConfig.IdpBackendAddress,
			IdpIssuerAddress:   authConfig.IdpIssuerAddress,
			HubBackendAddress:  authConfig.HubBackendAddress,
			APIKeyClientID:     authConfig.APIKeyClientID,
		},
		APIKeyAccess: &sessionstore.APIKey{
			ClientID: clientID,
			Secret:   secret,
		},
	}

	if err := retrieveAccessTokenWithAPIKey(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to authenticate with API key: %w", err)
	}

	return session, nil
}

// retrieveAccessTokenWithAPIKey gets API key access tokens.
func retrieveAccessTokenWithAPIKey(ctx context.Context, session *sessionstore.HubSession) error {
	if session == nil || !HasAPIKey(session) {
		return errors.New("no API key access available in the session")
	}

	idpClient := idp.NewClient(session.AuthConfig.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient(), session.AuthConfig.APIKeyClientID)

	var err error

	var idpResp *idp.GetAccessTokenResponse
	if utils.IsIAMAuthConfig(session) {
		idpResp, err = idpClient.GetAccessTokenFromAPIKey(ctx, session.APIKeyAccess.ClientID, session.APIKeyAccess.Secret)
		if err != nil {
			return fmt.Errorf("failed to refresh API key access token: %w", err)
		}
	} else {
		idpResp, err = idpClient.GetAccessTokenFromOkta(ctx, session.APIKeyAccess.ClientID, session.APIKeyAccess.Secret)
		if err != nil {
			return fmt.Errorf("failed to refresh API key access token: %w", err)
		}
	}

	session.APIKeyAccessToken = &sessionstore.Tokens{
		AccessToken:  idpResp.AccessToken,
		RefreshToken: idpResp.RefreshToken,
		IDToken:      idpResp.IDToken,
	}

	return nil
}
