// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaim_GetPayload_Roles(t *testing.T) {
	tests := []struct {
		name string
		role ClaimRole
		want string
	}{
		{
			name: "unspecified",
			role: ClaimRole_CLAIM_ROLE_UNSPECIFIED,
			want: `{"record_cid":"cid","role":"CLAIM_ROLE_UNSPECIFIED","subject":"dns:acme.com","signed_at":"2026-09-29T00:00:00Z","expires_at":""}`,
		},
		{
			name: "identity",
			role: ClaimRole_CLAIM_ROLE_IDENTITY,
			want: `{"record_cid":"cid","role":"CLAIM_ROLE_IDENTITY","subject":"dns:acme.com","signed_at":"2026-09-29T00:00:00Z","expires_at":""}`,
		},
		{
			name: "owner",
			role: ClaimRole_CLAIM_ROLE_OWNER,
			want: `{"record_cid":"cid","role":"CLAIM_ROLE_OWNER","subject":"dns:acme.com","signed_at":"2026-09-29T00:00:00Z","expires_at":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certificate := "cert"
			claim := &Claim{
				Role:        tt.role,
				RecordCid:   "cid",
				Subject:     "dns:acme.com",
				SignedAt:    "2026-09-29T00:00:00Z",
				Signature:   "sig",
				Certificate: &certificate,
			}

			got, err := claim.GetPayload()
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(got))
			require.Equal(t, tt.want, string(got), "field order must be stable")
		})
	}
}

func TestClaim_GetPayload_ExpiresAt(t *testing.T) {
	expiresAt := "2027-01-01T00:00:00Z"
	claim := &Claim{Role: ClaimRole_CLAIM_ROLE_IDENTITY, ExpiresAt: &expiresAt}

	got, err := claim.GetPayload()
	require.NoError(t, err)
	require.Contains(t, string(got), `"expires_at":"2027-01-01T00:00:00Z"`)
}
