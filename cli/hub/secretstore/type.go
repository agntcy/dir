// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package secretstore

type HubSecrets struct {
	HubSecrets map[string]*HubSecret `json:"hubSecrets"`
}
type HubSecret struct {
	*AuthConfig  `json:"auth_config"`
	*TokenSecret `json:"tokens"`
}

type TokenSecret struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type AuthConfig struct {
	ClientID           string `json:"client_id"`
	ProductID          string `json:"product_id"`
	IdpFrontendAddress string `json:"idp_frontend"`
	IdpBackendAddress  string `json:"idp_backend"`
	IdpIssuerAddress   string `json:"idp_issuer"`
	HubBackendAddress  string `json:"hub_backend"`
}
