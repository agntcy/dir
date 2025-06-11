// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package token provides utilities for working with JWT tokens, including validation and expiration checks.
package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// IsTokenExpired returns true if the given JWT access token is expired or invalid.
func IsTokenExpired(token string) bool {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, &claims); err != nil {
		return true
	}

	expTime, err := claims.GetExpirationTime()
	if err != nil || expTime == nil || expTime.Before(time.Now()) {
		return true
	}

	return false
}
