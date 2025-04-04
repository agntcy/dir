// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/docker/docker/builder/remotecontext/urlutil"
)

var (
	ErrInvalidFrontendURL = errors.New("invalid frontend URL")
	ErrFetchingConfig     = errors.New("error fetching config")
	ErrParsingConfig      = errors.New("error parsing config")
)

type AuthConfig struct {
	IdpProductID       string `json:"IAM_PRODUCT_ID"`
	ClientID           string `json:"IAM_OIDC_CLIENT_ID"`
	IdpIssuerAddress   string `json:"IAM_OIDC_ISSUER"`
	IdpBackendAddress  string `json:"IAM_API"`
	IdpFrontendAddress string `json:"IAM_UI"`
	HubBackendAddress  string `json:"HUB_API"`
}

func FetchAuthConfig(frontedURL string) (*AuthConfig, error) {
	if !urlutil.IsURL(frontedURL) {
		return nil, ErrInvalidFrontendURL
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, frontedURL+"/config.json", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %w: (%d) %s", ErrFetchingConfig, errors.New("unexpected status code"), resp.StatusCode, resp.Body)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}

	var authConfig AuthConfig
	if err = json.Unmarshal(body, &authConfig); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParsingConfig, err)
	}

	backendAddr := authConfig.HubBackendAddress
	backendAddr = strings.TrimPrefix(backendAddr, "http://")
	backendAddr = strings.TrimPrefix(backendAddr, "https://")
	backendAddr = strings.TrimSuffix(backendAddr, "/")
	backendAddr = strings.TrimSuffix(backendAddr, "/v1alpha1")
	backendAddr = fmt.Sprintf("%s:%d", backendAddr, 443)
	authConfig.HubBackendAddress = backendAddr

	return &authConfig, nil
}
