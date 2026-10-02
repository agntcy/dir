// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"testing"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/authn"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// The interceptor authorizes a request on the caller's full SPIFFE ID, as the
// authn interceptor put it in the context.
func TestInterceptor_RegistryCredentials(t *testing.T) {
	t.Parallel()

	intercept := NewInterceptor(newPeerIdentityAuthorizer(t))

	tests := []struct {
		name string
		id   string // empty: not authenticated
		code codes.Code
	}{
		{"user in the node's own trust domain", "spiffe://example.org/ns/dir/sa/dirctl", codes.PermissionDenied},
		{"named peer node", "spiffe://partner.org/ns/dir/sa/dir", codes.OK},
		{"no SPIFFE ID", "", codes.Unauthenticated},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			if tc.id != "" {
				ctx = context.WithValue(ctx, authn.SpiffeIDContextKey, spiffeid.RequireFromString(tc.id))
			}

			err := intercept(ctx, storev1.SyncService_RequestRegistryCredentials_FullMethodName)
			assert.Equal(t, tc.code, status.Code(err))
		})
	}
}
