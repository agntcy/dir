// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package idp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type TenantListResponse struct {
	Tenants []*TenantResponse `json:"tenants"`
}

type TenantResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	OrganizationID string   `json:"organizationId"`
	Entitlements   []string `json:"entitlements"`
}

type GetTenantsInProductResponse struct {
	TenantList *TenantListResponse
	Response   *http.Response
	Body       []byte
}

type RequestModifier func(*http.Request) error

func WithBearerToken(token string) RequestModifier {
	return func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)

		return nil
	}
}

type Client interface {
	GetTenantsInProduct(productID string, modifier ...RequestModifier) (*GetTenantsInProductResponse, error)
}
type client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string, httpClient *http.Client) Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *client) GetTenantsInProduct(productID string, modifiers ...RequestModifier) (*GetTenantsInProductResponse, error) {
	const path = "/v1alpha1/tenant"

	params := url.Values{}
	params.Set("product", productID)

	requestURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, requestURL, nil)
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
