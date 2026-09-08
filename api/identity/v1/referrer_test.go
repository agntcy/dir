// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityClaim_MarshalUnmarshalReferrer_RoundTrip(t *testing.T) {
	claim := identityv1.NewIdentityClaim("did:web:acme.com:agents:finance")
	claim.SignedAt = "2026-09-08T12:00:00Z"
	claim.Signature = "c2ln"

	ref, err := claim.MarshalReferrer()
	require.NoError(t, err)
	assert.Equal(t, corev1.IdentityClaimReferrerType, ref.GetType())

	var decoded identityv1.IdentityClaim
	require.NoError(t, decoded.UnmarshalReferrer(ref))
	assert.Equal(t, claim.GetSubject(), decoded.GetSubject())
	assert.Equal(t, claim.GetSignature(), decoded.GetSignature())
}

func TestOwnershipClaim_MarshalUnmarshalReferrer_RoundTrip(t *testing.T) {
	claim := identityv1.NewOwnershipClaim("did:web:acme.com")
	claim.SignedAt = "2026-09-08T12:00:00Z"
	claim.Signature = "c2ln"

	ref, err := claim.MarshalReferrer()
	require.NoError(t, err)
	assert.Equal(t, corev1.OwnershipClaimReferrerType, ref.GetType())

	var decoded identityv1.OwnershipClaim
	require.NoError(t, decoded.UnmarshalReferrer(ref))
	assert.Equal(t, claim.GetSubject(), decoded.GetSubject())
	assert.Equal(t, claim.GetSignature(), decoded.GetSignature())
}

func TestUnmarshalReferrer_NilData(t *testing.T) {
	var claim identityv1.IdentityClaim
	err := claim.UnmarshalReferrer(&corev1.RecordReferrer{Type: corev1.IdentityClaimReferrerType})
	require.Error(t, err)
}
