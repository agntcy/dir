// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package claimtest holds the end-to-end flow the resolver tests share:
// sign a claim, resolve its subject's keys, verify the claim against them.
// It lives apart from testutil because it imports identity and jws.
package claimtest

import (
	"crypto"
	"encoding/base64"
	"fmt"
	"testing"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/internal/testutil"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/stretchr/testify/require"
)

// RecordCID is the record every test claim is attached to.
const RecordCID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

// SignedClaim signs a fresh identity claim for subject the way a publisher would.
func SignedClaim(t *testing.T, subject string, key crypto.PrivateKey) *identityv1.Claim {
	t.Helper()

	signer, err := jws.NewKeySigner(testutil.PKCS8PEM(t, key), nil)
	require.NoError(t, err)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: subject}
	require.NoError(t, identity.Sign(claim, RecordCID, signer))

	return claim
}

// SignedCertClaim signs a claim for subject that carries certPEM, the certificate of key.
func SignedCertClaim(t *testing.T, subject string, key crypto.PrivateKey, certPEM []byte) *identityv1.Claim {
	t.Helper()

	signer, err := jws.NewKeySigner(testutil.PKCS8PEM(t, key), nil)
	require.NoError(t, err)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: subject}
	require.NoError(t, identity.Sign(claim, RecordCID, signer, identity.WithCertificate(certPEM)))

	return claim
}

// Verify is the flow a real caller follows: resolve the subject's keys, then
// verify the claim against all of them.
func Verify(t *testing.T, r resolvers.Resolver, claim *identityv1.Claim) (bool, error) {
	t.Helper()

	var certificate []byte

	if claim.Certificate != nil {
		var err error

		certificate, err = base64.StdEncoding.DecodeString(claim.GetCertificate())
		require.NoError(t, err)
	}

	keys, err := r.Resolve(t.Context(), claim.GetSubject(), certificate)
	if err != nil {
		return false, fmt.Errorf("resolve: %w", err)
	}

	ok, err := identity.Verify(claim, RecordCID, claim.GetSubject(), keys...)
	if err != nil {
		return false, fmt.Errorf("verify: %w", err)
	}

	return ok, nil
}
