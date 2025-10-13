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
	httpUtils "github.com/agntcy/dir/hub/utils/http"
)

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
	}

	if err := retrieveAccessTokenWithAPIKey(ctx, session, clientID, secret); err != nil {
		return nil, fmt.Errorf("failed to authenticate with API key: %w", err)
	}

	return session, nil
}

// retrieveAccessTokenWithAPIKey gets API key access tokens.
func retrieveAccessTokenWithAPIKey(ctx context.Context, session *sessionstore.HubSession, clientID, secret string) error {
	if session == nil {
		return errors.New("no session provided")
	}

	idpClient := idp.NewClient(session.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient(), session.APIKeyClientID)

	var err error

	var idpResp *idp.GetAccessTokenResponse
	if utils.IsIAMAuthConfig(session) {
		idpResp, err = idpClient.GetAccessTokenFromAPIKey(ctx, clientID, secret)
		if err != nil {
			return fmt.Errorf("failed to retrieve API key access token: %w", err)
		}
	} else {
		idpResp, err = idpClient.GetAccessTokenFromOkta(ctx, clientID, secret)
		if err != nil {
			return fmt.Errorf("failed to retrieve API key access token: %w", err)
		}
	}

	session.Tokens = &sessionstore.Tokens{
		AccessToken:  idpResp.AccessToken,
		RefreshToken: idpResp.RefreshToken,
		IDToken:      idpResp.IDToken,
	}

	return nil
}
