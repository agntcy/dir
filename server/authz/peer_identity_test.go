// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"testing"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/authz/config"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authorizeCase struct {
	name      string
	id        string
	apiMethod string
	allow     bool
}

func newPeerIdentityAuthorizer(t *testing.T) *Authorizer {
	t.Helper()

	authorizer, err := NewAuthorizer(config.Config{
		EnforcerPolicyFilePath: "./testdata/peer_identity_policies.csv",
	})
	require.NoError(t, err)

	return authorizer
}

// assertAuthorized checks each case against the peer identity policies.
func assertAuthorized(t *testing.T, cases []authorizeCase) {
	t.Helper()

	authorizer := newPeerIdentityAuthorizer(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			allowed, err := authorizer.Authorize(spiffeid.RequireFromString(tc.id), tc.apiMethod)
			require.NoError(t, err)
			assert.Equal(t, tc.allow, allowed)
		})
	}
}

// Registry credentials read every record without passing through the server,
// so only the peer nodes a rule names by SPIFFE ID obtain them. The policies
// also grant them to every trust domain, and the node's own trust domain
// everything: neither rule counts.
func TestAuthorize_RegistryCredentialsNeedPeerIdentity(t *testing.T) {
	t.Parallel()

	credentials := storev1.SyncService_RequestRegistryCredentials_FullMethodName

	assertAuthorized(t, []authorizeCase{
		{"user in the node's own trust domain", "spiffe://example.org/ns/dir/sa/dirctl", credentials, false},
		{"named peer node", "spiffe://partner.org/ns/dir/sa/dir", credentials, true},
		{"other workload whose ID starts with the peer's", "spiffe://partner.org/ns/dir/sa/dirctl", credentials, false},
		{"workload under a peer prefix", "spiffe://example.org/ns/peers/node-b", credentials, true},
		{"caller no rule names", "spiffe://other.org/ns/dir/sa/dir", credentials, false},
	})
}

// Trust-domain rules grant every other method as before.
func TestAuthorize_TrustDomainRulesStillApply(t *testing.T) {
	t.Parallel()

	assertAuthorized(t, []authorizeCase{
		{"own trust domain, any method", "spiffe://example.org/ns/dir/sa/dirctl", storev1.StoreService_Push_FullMethodName, true},
		{"any trust domain, a method granted to all", "spiffe://other.org/ns/dir/sa/dir", storev1.StoreService_Pull_FullMethodName, true},
		{"any trust domain, a method not granted", "spiffe://other.org/ns/dir/sa/dir", storev1.StoreService_Push_FullMethodName, false},
		{"peer rule grants only its method", "spiffe://partner.org/ns/dir/sa/dir", storev1.StoreService_Push_FullMethodName, false},
	})
}
