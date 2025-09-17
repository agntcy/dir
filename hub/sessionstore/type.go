// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package sessionstore provides session and token storage for the Agent Hub CLI and related applications.
package sessionstore

// ContextKey is a type for context keys used in session management.
type ContextKey string

// SessionContextKey is the key for storing/retrieving session in cmd.Context().
var SessionContextKey ContextKey = "session"

// HubSessions holds a map of session keys to HubSession objects.
type HubSessions struct {
	HubSessions map[string]*HubSession `json:"hub_sessions"`
}

// HubSession represents a user session, including tokens, current organization, user, and auth config.
type HubSession struct {
	Tokens      *Tokens `json:"tokens"`
	User        string  `json:"user"`
	*AuthConfig `json:"auth_config,omitempty"`
}

// Tokens holds ID, refresh, and access tokens for a session.
type Tokens struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// AuthConfig contains authentication and backend configuration for a session.
type AuthConfig struct {
	ClientID           string `json:"client_id"`
	IdpProductID       string `json:"product_id"`
	IdpFrontendAddress string `json:"idp_frontend"`
	IdpBackendAddress  string `json:"idp_backend"`
	IdpIssuerAddress   string `json:"idp_issuer"`
	HubBackendAddress  string `json:"hub_backend"`
	APIKeyClientID     string `json:"api_key_client_id"`
}
