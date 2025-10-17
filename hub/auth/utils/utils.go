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

	return ctx
}

func IsIAMAuthConfig(currentSession *sessionstore.HubSession) bool {
	if currentSession == nil || currentSession.AuthConfig == nil {
		return false
	}

	if currentSession.IdpFrontendAddress != "" && currentSession.IdpBackendAddress != "" {
		return true
	}

	return false
}
