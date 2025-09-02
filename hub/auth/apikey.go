// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/agntcy/dir/hub/auth/utils"
	"github.com/agntcy/dir/hub/client/idp"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/file"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
)

func RefreshAPIKeyAccessToken(ctx context.Context, session *sessionstore.HubSession, sessionKey string) error {
	// If the session has an API key access token, we can refresh it.
	if session == nil || !HasAPIKey(session) {
		return errors.New("no API key access token available in the session")
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

	// Load session store for saving
	sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())

	// Save updated session
	if err := sessionStore.SaveHubSession(sessionKey, session); err != nil {
		return fmt.Errorf("failed to save sessions: %w", err)
	}

	return nil
}
