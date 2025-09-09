// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package idp provides a client for interacting with the Identity Provider (IDP) API, including organization management.
package idp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// Path to the IDP API for retrieving an AccessToken for ApiKey Access.
	AccessTokenEndpoint = "/v1/token"
	// Scope for the access token request.
	AccessTokenScope = "openid offline_access"
	// Grant type for the access token request.
	AccessTokenGrantType = "password"
	// AccessTokenGrantTypeClientCredentials is the grant type for client credentials.
	AccessTokenGrantTypeClientCredentials = "client_credentials"
	// Custom scope for the access token request.
	AccessTokenCustomScope = "customScope"
)

// GetAccessTokenResponse contains the response when requesting an access token from the IDP API.
type GetAccessTokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
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
	// GetAccessToken retrieves an access token using the provided username and secret.
	GetAccessTokenFromAPIKey(ctx context.Context, username string, secret string) (*GetAccessTokenResponse, error)

	// GetAccessToken retrieves an access token using the provided username and secret.
	GetAccessTokenFromOkta(ctx context.Context, username string, secret string) (*GetAccessTokenResponse, error)
}

// client implements the Client interface for the IDP API.
type client struct {
	httpClient     *http.Client
	baseURL        string
	apiKeyClientID string
}

// NewClient creates a new IDP client with the given base URL and HTTP client.
func NewClient(baseURL string, httpClient *http.Client, apiKeyClientID string) Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{
		baseURL:        baseURL,
		httpClient:     httpClient,
		apiKeyClientID: apiKeyClientID,
	}
}

func (c *client) GetAccessTokenFromAPIKey(ctx context.Context, username string, base64Secret string) (*GetAccessTokenResponse, error) {
	requestURL := fmt.Sprintf("%s%s", c.baseURL, AccessTokenEndpoint)

	// Decode the base64 secret to get the plain text original secret.
	secretBytes, err := base64.StdEncoding.DecodeString(base64Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 secret: %w", err)
	}

	// IAM expects unescaped secret
	data := url.Values{}
	data.Set("grant_type", AccessTokenGrantType)
	data.Set("username", username)
	data.Set("password", string(secretBytes))
	data.Set("scope", AccessTokenScope)
	data.Set("client_id", c.apiKeyClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return handleTokenResponse(resp)
}

/*
This is based on https://developer.okta.com/docs/guides/implement-grant-type/clientcreds/main/#request-for-token
*/
func (c *client) GetAccessTokenFromOkta(ctx context.Context, clientID string, base64Secret string) (*GetAccessTokenResponse, error) {
	// The request URL for the Okta API to get an access token.
	requestURL := fmt.Sprintf("%s%s", c.baseURL, AccessTokenEndpoint)

	// Decode the base64 secret to concatenate with client id.
	secretBytes, err := base64.StdEncoding.DecodeString(base64Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 secret: %w", err)
	}

	// The auth header for the request, which includes the client ID and secret.
	authToken := fmt.Sprintf("%s:%s", clientID, string(secretBytes))
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(authToken))

	// The body for the request, which includes the grant type and scope.
	data := url.Values{}
	data.Set("grant_type", AccessTokenGrantTypeClientCredentials)
	data.Set("scope", AccessTokenCustomScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return handleTokenResponse(resp)
}

func handleTokenResponse(resp *http.Response) (*GetAccessTokenResponse, error) {
	var body []byte

	if resp.Body != nil {
		var err error

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
	}

	if resp.StatusCode == http.StatusOK {
		var parsedResponse *GetAccessTokenResponse
		if err := json.Unmarshal(body, &parsedResponse); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		return parsedResponse, nil
	}

	return nil, fmt.Errorf("bad response status code: %d, body: %s", resp.StatusCode, string(body))
}
