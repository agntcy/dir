// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/credentials"
)

// jwtPerRPCCredentials implements credentials.PerRPCCredentials for JWT authentication.
type jwtPerRPCCredentials struct {
	jwtSource *workloadapi.JWTSource
	audience  string
}

// GetRequestMetadata gets the current JWT token and attaches it to the request metadata.
func (c *jwtPerRPCCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	// Fetch JWT-SVID for the configured audience
	jwtSVID, err := c.jwtSource.FetchJWTSVID(ctx, jwtsvid.Params{
		Audience: c.audience,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWT-SVID: %w", err)
	}

	// Return the token as a Bearer token in the authorization header
	return map[string]string{
		"authorization": "Bearer " + jwtSVID.Marshal(),
	}, nil
}

// Returns true because JWT-SVID authentication should be used over TLS.
func (c *jwtPerRPCCredentials) RequireTransportSecurity() bool {
	return true
}

// newJWTCredentials creates a new PerRPCCredentials that injects JWT-SVIDs.
func newJWTCredentials(jwtSource *workloadapi.JWTSource, audience string) credentials.PerRPCCredentials {
	return &jwtPerRPCCredentials{
		jwtSource: jwtSource,
		audience:  audience,
	}
}
