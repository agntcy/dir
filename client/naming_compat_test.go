// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetVerificationInfo(t *testing.T) {
	verifiedAt := timestamppb.New(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	failure := "key mismatch"

	tests := []struct {
		name         string
		status       *identityv1.GetIdentityStatusResponse
		wantVerified bool
		wantDomain   string
		wantError    string
	}{
		{
			name: "verified owner",
			status: &identityv1.GetIdentityStatusResponse{Owner: &identityv1.ClaimVerification{
				Subject:    "dns:acme.com",
				Status:     identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED,
				VerifiedAt: verifiedAt,
			}},
			wantVerified: true,
			wantDomain:   "dns:acme.com",
		},
		{
			name: "failed owner reports the failure",
			status: &identityv1.GetIdentityStatusResponse{Owner: &identityv1.ClaimVerification{
				Subject: "dns:acme.com",
				Status:  identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_FAILED,
				Error:   &failure,
			}},
			wantError: failure,
		},
		{
			name:      "no owner claim",
			status:    &identityv1.GetIdentityStatusResponse{},
			wantError: "no verification found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{IdentityServiceClient: &fakeIdentityClient{statusResp: tt.status}}

			byCID, err := c.GetVerificationInfo(t.Context(), identityTestCID)
			require.NoError(t, err)

			byName, err := c.GetVerificationInfoByName(t.Context(), "acme.com/agent", "1.0.0")
			require.NoError(t, err)

			for _, got := range []interface {
				GetVerified() bool
				GetErrorMessage() string
			}{byCID, byName} {
				assert.Equal(t, tt.wantVerified, got.GetVerified())
				assert.Equal(t, tt.wantError, got.GetErrorMessage())
			}

			assert.Equal(t, tt.wantDomain, byCID.GetVerification().GetDomain().GetDomain())

			if tt.wantVerified {
				assert.Equal(t, verifiedAt.AsTime(), byCID.GetVerification().GetDomain().GetVerifiedAt().AsTime())
			}
		})
	}
}

func TestGetVerificationInfoByName_RequestsNameAndVersion(t *testing.T) {
	fake := &fakeIdentityClient{statusResp: &identityv1.GetIdentityStatusResponse{}}
	c := &Client{IdentityServiceClient: fake}

	_, err := c.GetVerificationInfoByName(t.Context(), "acme.com/agent", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "acme.com/agent", fake.statusReq.GetName())
	assert.Equal(t, "1.0.0", fake.statusReq.GetVersion())
	assert.Nil(t, fake.statusReq.Cid)

	_, err = c.GetVerificationInfoByName(t.Context(), "", "")
	require.Error(t, err)

	_, err = c.GetVerificationInfo(t.Context(), "")
	require.Error(t, err)
}
