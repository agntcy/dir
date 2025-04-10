// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sessionstore

type HubSessions struct {
	HubSessions map[string]*HubSession `json:"hub_sessions"`
}
type HubSession struct {
	Tokens        map[string]*Tokens `json:"tokens"`
	CurrentTenant string             `json:"current_tenant"`
	User          string             `json:"user"`
	*AuthConfig   `json:"auth_config,omitempty"`
}

type Tokens struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// TODO: AuthConfig should be cahed later and stored in the session. That's why object in the session store.
type AuthConfig struct {
	ClientID           string `json:"client_id"`
	IdpProductID       string `json:"product_id"`
	IdpFrontendAddress string `json:"idp_frontend"`
	IdpBackendAddress  string `json:"idp_backend"`
	IdpIssuerAddress   string `json:"idp_issuer"`
	HubBackendAddress  string `json:"hub_backend"`
}
