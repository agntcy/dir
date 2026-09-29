// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/require"
)

func TestClaim_MarshalUnmarshalReferrer_RoundTrip(t *testing.T) {
	claim := &Claim{Role: ClaimRole_CLAIM_ROLE_OWNER, Subject: "dns:acme.com"}
	claim.RecordCid = "baguqeera"
	claim.SignedAt = "2026-09-29T00:00:00Z"

	ref, err := claim.MarshalReferrer()
	require.NoError(t, err)
	require.Equal(t, corev1.OwnershipClaimReferrerType, ref.GetType())

	var got Claim
	require.NoError(t, got.UnmarshalReferrer(ref))

	require.Equal(t, claim.GetRole(), got.GetRole())
	require.Equal(t, claim.GetRecordCid(), got.GetRecordCid())
	require.Equal(t, claim.GetSubject(), got.GetSubject())
	require.Equal(t, claim.GetSignedAt(), got.GetSignedAt())
}

func TestClaim_MarshalReferrer_TypeByRole(t *testing.T) {
	identity, err := (&Claim{Role: ClaimRole_CLAIM_ROLE_IDENTITY, Subject: "dns:acme.com"}).MarshalReferrer()
	require.NoError(t, err)
	require.Equal(t, corev1.IdentityClaimReferrerType, identity.GetType())

	owner, err := (&Claim{Role: ClaimRole_CLAIM_ROLE_OWNER, Subject: "dns:acme.com"}).MarshalReferrer()
	require.NoError(t, err)
	require.Equal(t, corev1.OwnershipClaimReferrerType, owner.GetType())
}

func TestClaim_MarshalReferrer_Nil(t *testing.T) {
	var claim *Claim

	_, err := claim.MarshalReferrer()
	require.Error(t, err)
}

func TestClaim_UnmarshalReferrer_NilData(t *testing.T) {
	var claim Claim

	require.Error(t, claim.UnmarshalReferrer(nil))
	require.Error(t, claim.UnmarshalReferrer(&corev1.RecordReferrer{}))
}
