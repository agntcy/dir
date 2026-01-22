// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	"google.golang.org/grpc/credentials"
)

// githubPerRPCCredentials implements credentials.PerRPCCredentials for GitHub OAuth2 token authentication.
type githubPerRPCCredentials struct {
	token string
}

// GetRequestMetadata attaches the GitHub OAuth2 token to the request metadata as a Bearer token.
func (c *githubPerRPCCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + c.token,
	}, nil
}

// RequireTransportSecurity returns false because GitHub auth can work over insecure connections
// (the Envoy gateway handles TLS termination externally).
func (c *githubPerRPCCredentials) RequireTransportSecurity() bool {
	return false
}

// newGitHubCredentials creates a new PerRPCCredentials that injects a GitHub OAuth2 token.
func newGitHubCredentials(token string) credentials.PerRPCCredentials {
	return &githubPerRPCCredentials{
		token: token,
	}
}
