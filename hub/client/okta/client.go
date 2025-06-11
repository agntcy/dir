// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package okta provides a client for interacting with the Okta identity provider, including OAuth flows, token management, and logout operations.
package okta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	paramClientID            = "client_id"
	paramCodeChallenge       = "code_challenge"
	paramCodeChallengeMethod = "code_challenge_method"
	paramNonce               = "nonce"
	paramRedirectURI         = "redirect_uri"
	paramResponseType        = "response_type"
	paramState               = "state"
	paramScope               = "scope"
	paramGrantType           = "grant_type"
	paramCode                = "code"
	paramCodeVerifier        = "code_verifier"
	paramRefreshToken        = "refresh_token"

	headerAccept       = "Accept"
	headerContentType  = "Content-Type"
	headerCacheControl = "Cache-Control"
)

var (
	ErrRequestCreation = errors.New("failed to create request")
	ErrRequestSending  = errors.New("failed to send request")
	ErrParsingResponse = errors.New("failed to parse response")
)

// RequestTokenRequest contains parameters for exchanging an authorization code for tokens.
type RequestTokenRequest struct {
	ClientID    string
	RedirectURI string
	Verifier    string
	Code        string
}

// Token represents OAuth tokens returned by Okta.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

// RequestTokenResponse contains the response from the token endpoint.
type RequestTokenResponse struct {
	Token    *Token
	Body     []byte
	Response *http.Response
}

// ChallengeMethod represents the PKCE challenge method type.
type ChallengeMethod string

const (
	ChallengeMethodS256 ChallengeMethod = "S256"
)

// AuthorizeRequest contains parameters for building the authorization URL.
type AuthorizeRequest struct {
	ClientID      string
	S256Challenge string
	Nonce         string
	RedirectURI   string
	RequestID     string
}

// LogoutRequest contains parameters for logging out of Okta.
type LogoutRequest struct {
	IDToken string
}

// LogoutResponse contains the response from the logout endpoint.
type LogoutResponse struct {
	Body     []byte
	Response *http.Response
}

// RefreshTokenRequest contains parameters for refreshing tokens.
type RefreshTokenRequest struct {
	ClientID     string
	RefreshToken string
}

// RefreshTokenResponse contains the response from the refresh token endpoint.
type RefreshTokenResponse struct {
	Response *http.Response
	Body     []byte
	Token    *Token
	Error    *ErrorResponse
}

// ErrorResponse represents an error returned by Okta.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// Client defines the interface for interacting with Okta for OAuth and token operations.
type Client interface {
	// AuthorizeURL constructs the authorization URL for the OAuth flow.
	AuthorizeURL(r *AuthorizeRequest) string

	// RequestToken exchanges an authorization code for tokens.
	RequestToken(request *RequestTokenRequest) (*RequestTokenResponse, error)
	// Logout revokes the current session's ID token.
	Logout(request *LogoutRequest) (*LogoutResponse, error)
	// RefreshToken exchanges a refresh token for new tokens.
	RefreshToken(*RefreshTokenRequest) (*RefreshTokenResponse, error)
}

// IdpClient implements the Client interface for Okta.
type IdpClient struct {
	BaseURL string

	httpClient *http.Client
}

// NewClient creates a new Okta client with the given base URL and HTTP client.
func NewClient(baseURL string, httpClient *http.Client) *IdpClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &IdpClient{
		BaseURL:    baseURL,
		httpClient: httpClient,
	}
}

// RequestToken exchanges an authorization code for tokens.
func (i *IdpClient) RequestToken(request *RequestTokenRequest) (*RequestTokenResponse, error) {
	data := url.Values{}
	data.Set(paramGrantType, "authorization_code")
	data.Set(paramClientID, request.ClientID)
	data.Set(paramRedirectURI, request.RedirectURI)
	data.Set(paramCode, request.Code)
	data.Set(paramCodeVerifier, request.Verifier)

	tokenReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, i.BaseURL+"/v1/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: token request", ErrRequestCreation)
	}

	tokenReq.Header.Add(headerAccept, "application/json")
	tokenReq.Header.Add(headerContentType, "application/x-www-form-urlencoded")
	tokenReq.Header.Add(headerCacheControl, "no-cache")

	resp, err := i.httpClient.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("%w: token request", ErrRequestSending)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrParsingResponse
	}

	if resp.StatusCode == http.StatusOK {
		var t *Token

		err = json.Unmarshal(body, &t)
		if err != nil {
			return nil, ErrParsingResponse
		}

		return &RequestTokenResponse{
			Token:    t,
			Body:     body,
			Response: resp,
		}, nil
	}

	return &RequestTokenResponse{
		Response: resp,
		Body:     body,
	}, nil
}

// AuthorizeURL constructs the authorization URL for the OAuth flow.
func (i *IdpClient) AuthorizeURL(r *AuthorizeRequest) string {
	params := url.Values{}
	params.Add(paramClientID, r.ClientID)
	params.Add(paramCodeChallenge, r.S256Challenge)
	params.Add(paramCodeChallengeMethod, string(ChallengeMethodS256))
	params.Add(paramNonce, r.Nonce)
	params.Add(paramRedirectURI, r.RedirectURI)
	params.Add(paramResponseType, "code")
	params.Add(paramState, fmt.Sprintf(`{"sessionRequest":"%s"}`, r.RequestID))
	params.Add(paramScope, "openid offline_access")

	return fmt.Sprintf("%s/v1/authorize?%s", i.BaseURL, params.Encode())
}

// Logout revokes the current session's ID token.
func (i *IdpClient) Logout(request *LogoutRequest) (*LogoutResponse, error) {
	data := url.Values{}
	data.Set("id_token_hint", request.IDToken)

	logoutReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, i.BaseURL+"/v1/logout", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: logout request", ErrRequestCreation)
	}

	logoutReq.Header.Add(headerAccept, "application/json")
	logoutReq.Header.Add(headerContentType, "application/x-www-form-urlencoded")
	logoutReq.Header.Add(headerCacheControl, "no-cache")

	resp, err := i.httpClient.Do(logoutReq)
	if err != nil {
		return nil, fmt.Errorf("%w: logout request", ErrRequestSending)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrParsingResponse
	}

	return &LogoutResponse{
		Body:     body,
		Response: resp,
	}, nil
}

// RefreshToken exchanges a refresh token for new tokens.
func (i *IdpClient) RefreshToken(req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	data := url.Values{}
	data.Set(paramGrantType, "refresh_token")
	data.Set(paramClientID, req.ClientID)
	data.Set(paramRefreshToken, req.RefreshToken)

	tokenReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, i.BaseURL+"/v1/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: token request", ErrRequestCreation)
	}

	tokenReq.Header.Add(headerAccept, "application/json")
	tokenReq.Header.Add(headerContentType, "application/x-www-form-urlencoded")
	tokenReq.Header.Add(headerCacheControl, "no-cache")

	resp, err := i.httpClient.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("%w: token request", ErrRequestSending)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrParsingResponse
	}

	if resp.StatusCode == http.StatusOK {
		var t *Token

		err = json.Unmarshal(body, &t)
		if err != nil {
			return nil, ErrParsingResponse
		}

		return &RefreshTokenResponse{
			Response: resp,
			Body:     body,
			Token:    t,
		}, nil
	}

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		var e *ErrorResponse

		err = json.Unmarshal(body, &e)
		if err != nil {
			return nil, ErrParsingResponse
		}

		return &RefreshTokenResponse{
			Response: resp,
			Body:     body,
			Error:    e,
		}, nil
	}

	return &RefreshTokenResponse{
		Response: resp,
		Body:     body,
	}, nil
}
