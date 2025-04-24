// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"
)

const (
	tenantNameClaim = "tenant_name"
	userClaim       = "sub"
)

func RefreshContextTokenIfExpired(cmd *cobra.Command, opts *options.HubOptions) error {
	sessionStore, ok := context.GetSessionStoreFromContext(cmd)
	if !ok {
		return errors.New("failed to get session store from context")
	}

	session, ok := context.GetCurrentHubSessionFromContext(cmd)
	if !ok {
		return errors.New("failed to get current session from context")
	}

	oktaClient, ok := context.GetOktaClientFromContext(cmd)
	if !ok {
		return errors.New("failed to get okta client from context")
	}

	return refreshTokenIfExpired(
		cmd,
		opts.ServerAddress,
		session,
		sessionStore,
		oktaClient,
	)
}

func refreshTokenIfExpired(cmd *cobra.Command, sessionKey string, session *sessionstore.HubSession, secretStore sessionstore.SessionStore, oktaClient okta.Client) error {
	if session == nil ||
		session.Tokens == nil ||
		session.CurrentTenant == "" ||
		session.Tokens[session.CurrentTenant] == nil ||
		session.Tokens[session.CurrentTenant].AccessToken == "" {
		return nil
	}

	accessToken := session.Tokens[session.CurrentTenant].AccessToken
	refreshToken := session.Tokens[session.CurrentTenant].RefreshToken

	if !IsTokenExpired(accessToken) {
		return nil
	}

	if refreshToken == "" {
		return errors.New("access token is expired and refresh token is empty")
	}

	resp, err := oktaClient.RefreshToken(&okta.RefreshTokenRequest{
		RefreshToken: refreshToken,
		ClientID:     session.ClientID,
	})
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to refresh token: %s", string(resp.Body))
	}

	newTokenSecret := &sessionstore.Tokens{
		AccessToken:  resp.Token.AccessToken,
		RefreshToken: resp.Token.RefreshToken,
		IDToken:      resp.Token.IDToken,
	}
	session.Tokens[session.CurrentTenant] = newTokenSecret

	// Update context with new token
	if ok := context.SetCurrentHubSessionForContext(cmd, session); !ok {
		return errors.New("failed to set current hub session for context")
	}

	// Update tokens store with new token
	if err = secretStore.SaveHubSession(sessionKey, session); err != nil {
		return fmt.Errorf("failed to save hub tokens: %w", err)
	}

	return nil
}

func GetTenantNameFromToken(token string) (string, error) {
	return getClaimFromToken(token, tenantNameClaim)
}

func GetUserFromToken(token string) (string, error) {
	return getClaimFromToken(token, userClaim)
}

func getClaimFromToken(token, claim string) (string, error) {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, &claims); err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	value, ok := claims[claim].(string)
	if !ok {
		return "", fmt.Errorf("failed to get %s from token", claim)
	}

	return value, nil
}
