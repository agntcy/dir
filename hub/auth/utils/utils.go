// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/agntcy/dir/hub/sessionstore"
	"google.golang.org/grpc/metadata"
)

// AddAuthToContext adds the authorization header to the context if an access token is available.
func AddAuthToContext(ctx context.Context, session *sessionstore.HubSession) context.Context {
	// Using login credential if available
	if session != nil && session.Tokens != nil {
		if t := session.Tokens; t != nil && t.AccessToken != "" {
			return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+t.AccessToken))
		}
	}
	// Otherwise, using API key access token if present
	if session != nil && session.APIKeyAccessToken != nil && session.APIKeyAccessToken.AccessToken != "" {
		return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+session.APIKeyAccessToken.AccessToken))
	}

	return ctx
}

func IsIAMAuthConfig(currentSession *sessionstore.HubSession) bool {
	if currentSession == nil || currentSession.AuthConfig == nil {
		return false
	}

	if currentSession.AuthConfig.IdpFrontendAddress != "" && currentSession.AuthConfig.IdpBackendAddress != "" {
		return true
	}

	return false
}
