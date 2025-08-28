// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package token provides utilities for working with JWT tokens, including validation, refresh, and claim extraction.
package token

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/golang-jwt/jwt/v5"
)

const (
	userClaim = "sub"
)

// RefreshTokenIfExpired refreshes the access token for the current session if it is expired.
// It uses the provided Okta client and session store to update the session and persist the new tokens.
// Returns an error if the refresh or save fails.
func RefreshTokenIfExpired(sessionKey string, session *sessionstore.HubSession, secretStore sessionstore.SessionStore, oktaClient okta.Client) error {
	if session == nil ||
		session.Tokens == nil ||
		session.Tokens.AccessToken == "" {
		return nil
	}

	accessToken := session.Tokens.AccessToken
	refreshToken := session.Tokens.RefreshToken

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
	session.Tokens = newTokenSecret

	// Update tokens store with new token
	if err = secretStore.SaveHubSession(sessionKey, session); err != nil {
		return fmt.Errorf("failed to save hub tokens: %w", err)
	}

	return nil
}

// GetUserFromToken extracts the user (subject) from the given JWT access token.
func GetUserFromToken(token string) (string, error) {
	return getClaimFromToken(token, userClaim)
}

// getClaimFromToken extracts a string claim from the given JWT access token.
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
