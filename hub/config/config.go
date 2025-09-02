// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration management for the Agent Hub CLI and related applications.
package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	httpUtils "github.com/agntcy/dir/hub/utils/http"
	urlutils "github.com/agntcy/dir/hub/utils/url"
)

var (
	ErrInvalidFrontendURL = errors.New("invalid frontend URL")
	ErrFetchingConfig     = errors.New("error fetching config")
	ErrParsingConfig      = errors.New("error parsing config")
)

// AuthConfig holds authentication and backend configuration values loaded from the IDP config endpoint.
type AuthConfig struct {
	IdpProductID       string `json:"IAM_PRODUCT_ID"`
	ClientID           string `json:"IAM_OIDC_CLIENT_ID"`
	IdpIssuerAddress   string `json:"IAM_OIDC_ISSUER"`
	IdpBackendAddress  string `json:"IAM_API"`
	IdpFrontendAddress string `json:"IAM_UI"`
	HubBackendAddress  string `json:"HUB_API"`
	APIKeyClientID     string `json:"API_KEY_CLIENT_ID"`
}

// FetchAuthConfig retrieves and parses the AuthConfig from the given frontend URL.
// It validates the URL, fetches the config.json, and normalizes backend addresses.
// Returns the AuthConfig or an error if the operation fails.
func FetchAuthConfig(ctx context.Context, frontendURL string) (*AuthConfig, error) {
	if err := urlutils.ValidateSecureURL(frontendURL); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFrontendURL, err)
	}

	configJSONURL, err := url.JoinPath(frontendURL, "config.json")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFrontendURL, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configJSONURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}

	resp, err := httpUtils.CreateSecureHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, errors.New("failed to communicate with idp provider"))
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}

	var authConfig *AuthConfig
	if err = json.Unmarshal(body, &authConfig); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParsingConfig, err)
	}

	if authConfig == nil {
		return nil, fmt.Errorf("%w: %w", ErrParsingConfig, errors.New("config is nil"))
	}

	backendAddr := authConfig.HubBackendAddress
	backendAddr = strings.TrimPrefix(backendAddr, "http://")
	backendAddr = strings.TrimPrefix(backendAddr, "https://")
	backendAddr = strings.TrimSuffix(backendAddr, "/")
	authConfig.HubBackendAddress = backendAddr

	idpBackendAddr := authConfig.IdpBackendAddress
	idpBackendAddr = strings.TrimSuffix(idpBackendAddr, "/")
	// FIXME: is this trim still necessary?
	idpBackendAddr = strings.TrimSuffix(idpBackendAddr, "/v1alpha1")
	authConfig.IdpBackendAddress = idpBackendAddr

	return authConfig, nil
}
