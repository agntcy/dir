// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package token provides utilities for working with JWT tokens, including validation and expiration checks.
package token

import (
	"errors"

	"github.com/agntcy/dir/hub/sessionstore"
)

// ValidateAccessToken checks if the current session has a valid access token.
// Returns an error if the token is missing or invalid.
func ValidateAccessToken(session *sessionstore.HubSession) error {
	if session == nil || session.Tokens == nil || session.Tokens.AccessToken == "" {
		return errors.New("invalid session token")
	}

	return nil
}
