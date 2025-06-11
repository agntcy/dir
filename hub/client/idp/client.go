// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package idp provides a client for interacting with the Identity Provider (IDP) API, including tenant and organization management.
package idp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// TenantListResponse represents a list of tenants returned by the IDP API.
type TenantListResponse struct {
	Tenants []*TenantResponse `json:"tenants"`
}

// TenantResponse represents a single tenant's information.
type TenantResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	OrganizationID string   `json:"organizationId"`
	Entitlements   []string `json:"entitlements"`
}

// GetTenantsInProductResponse contains the response from the GetTenantsInProduct API call.
type GetTenantsInProductResponse struct {
	TenantList *TenantListResponse
	Response   *http.Response
	Body       []byte
}

// RequestModifier is a function that modifies an HTTP request before it is sent.
type RequestModifier func(*http.Request) error

// WithBearerToken returns a RequestModifier that adds a Bearer token to the Authorization header.
func WithBearerToken(token string) RequestModifier {
	return func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)

		return nil
	}
}

// Client defines the interface for interacting with the IDP API.
type Client interface {
	// GetTenantsInProduct retrieves the list of tenants for a given product ID.
	GetTenantsInProduct(ctx context.Context, productID string, modifier ...RequestModifier) (*GetTenantsInProductResponse, error)
}

// client implements the Client interface for the IDP API.
type client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new IDP client with the given base URL and HTTP client.
func NewClient(baseURL string, httpClient *http.Client) Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// GetTenantsInProduct retrieves the list of tenants for the specified product ID from the IDP API.
// It applies any provided request modifiers (e.g., for authentication).
// Returns the response or an error if the request fails.
func (c *client) GetTenantsInProduct(ctx context.Context, productID string, modifiers ...RequestModifier) (*GetTenantsInProductResponse, error) {
	const path = "/v1alpha1/tenant"

	params := url.Values{}
	params.Set("product", productID)

	requestURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for _, m := range modifiers {
		if err = m(req); err != nil {
			return nil, fmt.Errorf("failed to modify request: %w", err)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var body []byte
	if resp.Body != nil {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
	}

	if resp.StatusCode == http.StatusOK {
		var parsedResponse *TenantListResponse
		if err = json.Unmarshal(body, &parsedResponse); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		return &GetTenantsInProductResponse{
			TenantList: parsedResponse,
			Response:   resp,
			Body:       body,
		}, nil
	}

	return &GetTenantsInProductResponse{
		TenantList: nil,
		Response:   resp,
		Body:       body,
	}, nil
}
