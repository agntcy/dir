// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/hub/auth/utils"
	"github.com/agntcy/dir/hub/client/idp"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/file"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
)

func RefreshApiKeyAccessToken(ctx context.Context, session *sessionstore.HubSession, sessionKey string) error {
	// If the session has an API key access token, we can refresh it.
	if session == nil || !HasApiKey(session) {
		return fmt.Errorf("no API key access token available in the session")
	}

	idpClient := idp.NewClient(session.AuthConfig.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient())

	var idpResp *idp.GetAccessTokenResponse
	var err error
	if utils.IsIAMAuthConfig(session) {
		idpResp, err = idpClient.GetAccessTokenFromApiKey(ctx, session.ApiKeyAccess.ClientID, session.ApiKeyAccess.Secret)
		if err != nil {
			return fmt.Errorf("failed to refresh API key access token: %w", err)
		}
	} else {
		idpResp, err = idpClient.GetAccessTokenFromOkta(ctx, session.ApiKeyAccess.ClientID, session.ApiKeyAccess.Secret)
		if err != nil {
			return fmt.Errorf("failed to refresh API key access token: %w", err)
		}
	}

	session.ApiKeyAccessToken = &sessionstore.Tokens{
		AccessToken:  idpResp.AccessToken,
		RefreshToken: idpResp.RefreshToken,
		IDToken:      idpResp.IDToken,
	}

	// Load session store for saving
	sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())

	// Save updated session
	if err := sessionStore.SaveHubSession(sessionKey, session); err != nil {
		return fmt.Errorf("failed to save sessions: %w", err)
	}

	return nil
}
