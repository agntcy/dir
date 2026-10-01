// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity_test

import (
	"crypto"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/internal/testutil"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/stretchr/testify/require"
)

const (
	testRecordCID   = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
	otherRecordCID  = "bafybeih25dih6ug3xtj73vswccw423b56qh4pj5awfj6oyf2u2x7cwaczy"
	testSubjectDNS  = "dns:acme.com"
	testSubjectDID  = "did:web:acme.com"
	testSubjectSVID = "spiffe://acme.com/agents/finance"
)

func keySigner(t *testing.T, key crypto.Signer) *jws.KeySigner {
	t.Helper()

	signer, err := jws.NewKeySigner(testutil.PKCS8PEM(t, key), nil)
	require.NoError(t, err)

	return signer
}

func TestSignVerify_RoundTrip(t *testing.T) {
	for _, kind := range []string{"ES256", "ES384", "EdDSA", "RS256"} {
		t.Run(kind, func(t *testing.T) {
			for _, role := range []identityv1.ClaimRole{identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, identityv1.ClaimRole_CLAIM_ROLE_OWNER} {
				key := testutil.NewKey(t, kind)

				claim := &identityv1.Claim{Role: role, Subject: testSubjectDNS}
				require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))
				require.NotEmpty(t, claim.GetSignature())
				require.Equal(t, testRecordCID, claim.GetRecordCid())
				require.Nil(t, claim.Certificate)

				ok, err := identity.Verify(claim, testRecordCID, testSubjectDNS, key.Public())
				require.NoError(t, err)
				require.True(t, ok)
			}
		})
	}
}

func TestSignVerify_WithCertificate(t *testing.T) {
	key := testutil.NewKey(t, "ES256")
	cert := testutil.SelfSignedCertPEM(t, key, testSubjectSVID)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key), identity.WithCertificate(cert)))
	require.NotEmpty(t, claim.GetCertificate())

	ok, err := identity.Verify(claim, testRecordCID, testSubjectSVID, key.Public())
	require.NoError(t, err)
	require.True(t, ok)

	// A raw DER certificate is accepted as well.
	block, _ := pem.Decode(cert)
	require.NotNil(t, block)

	claim = &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key), identity.WithCertificate(block.Bytes)))
	require.NotEmpty(t, claim.GetCertificate())
}

func TestSign_RejectsInvalidInput(t *testing.T) {
	signer := keySigner(t, testutil.NewKey(t, "ES256"))

	require.Error(t, identity.Sign(nil, testRecordCID, signer), "nil claim")
	require.Error(t, identity.Sign(&identityv1.Claim{Subject: testSubjectDNS}, testRecordCID, signer), "unspecified role")
	require.Error(t, identity.Sign(&identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_OWNER}, testRecordCID, signer), "empty subject")
	require.Error(t, identity.Sign(&identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_OWNER, Subject: testSubjectDNS}, testRecordCID, nil), "nil signer")
}

func TestVerify_TamperedPayload(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))

	claim.SignedAt = time.Now().Add(time.Hour).UTC().Format(time.RFC3339) // tamper after signing

	ok, err := identity.Verify(claim, testRecordCID, testSubjectDNS, key.Public())
	require.Error(t, err)
	require.False(t, ok)
}

func TestVerify_WrongKey(t *testing.T) {
	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, testutil.NewKey(t, "ES256"))))

	ok, err := identity.Verify(claim, testRecordCID, testSubjectDNS, testutil.NewKey(t, "ES256").Public())
	require.Error(t, err)
	require.False(t, ok)
}

func TestVerify_MultipleKeys(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))

	ok, err := identity.Verify(claim, testRecordCID, testSubjectDNS, testutil.NewKey(t, "EdDSA").Public(), key.Public())
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = identity.Verify(claim, testRecordCID, testSubjectDNS)
	require.Error(t, err)
	require.False(t, ok)
}

func TestVerify_WrongSubject(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))

	ok, err := identity.Verify(claim, testRecordCID, testSubjectDID, key.Public())
	require.False(t, ok)
	require.ErrorContains(t, err, "does not match")
}

func TestVerify_MismatchedRole(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))

	// Replay the same claim, relabeled as an ownership claim for the same subject/record.
	claim.Role = identityv1.ClaimRole_CLAIM_ROLE_OWNER

	ok, err := identity.Verify(claim, testRecordCID, testSubjectDNS, key.Public())
	require.Error(t, err)
	require.False(t, ok)
}

func TestVerify_Expired(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	expired := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	claim.ExpiresAt = &expired
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))

	ok, err := identity.Verify(claim, testRecordCID, testSubjectDNS, key.Public())
	require.False(t, ok)
	require.ErrorContains(t, err, "expired")
}

func TestVerify_WrongRecordCID(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))

	ok, err := identity.Verify(claim, otherRecordCID, testSubjectDNS, key.Public())
	require.False(t, ok)
	require.ErrorContains(t, err, "record_cid")
}

type failingSigner struct{}

func (failingSigner) Sign([]byte) (string, error) { return "", errors.New("hsm unavailable") }

func (failingSigner) Public() crypto.PublicKey { return nil }

// requireUntouched asserts Sign left every field it manages as it found it.
func requireUntouched(t *testing.T, claim *identityv1.Claim) {
	t.Helper()

	require.Empty(t, claim.GetRecordCid())
	require.Empty(t, claim.GetSignedAt())
	require.Empty(t, claim.GetSignature())
	require.Nil(t, claim.Certificate)
}

func TestSign_FailureLeavesClaimUntouched(t *testing.T) {
	key := testutil.NewKey(t, "ES256")
	signer := keySigner(t, key)
	cert := testutil.SelfSignedCertPEM(t, key, testSubjectSVID)

	tests := map[string]struct {
		claim   *identityv1.Claim
		signer  jws.Signer
		opts    []identity.SignOption
		wantErr string
	}{
		"signer fails": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS},
			signer:  failingSigner{},
			wantErr: "hsm unavailable",
		},
		"subject not covered": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: "spiffe://acme.com/agents/other"},
			signer:  signer,
			opts:    []identity.SignOption{identity.WithCertificate(cert)},
			wantErr: "does not cover",
		},
		"certificate of another key": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID},
			signer:  keySigner(t, testutil.NewKey(t, "ES256")),
			opts:    []identity.SignOption{identity.WithCertificate(cert)},
			wantErr: "does not match",
		},
		"certificate of another key type": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID},
			signer:  keySigner(t, testutil.NewKey(t, "EdDSA")),
			opts:    []identity.SignOption{identity.WithCertificate(cert)},
			wantErr: "does not match",
		},
		"empty certificate": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID},
			signer:  signer,
			opts:    []identity.SignOption{identity.WithCertificate(nil)},
			wantErr: "parse certificate",
		},
		"garbage certificate": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID},
			signer:  signer,
			opts:    []identity.SignOption{identity.WithCertificate([]byte("not a certificate"))},
			wantErr: "parse certificate",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := identity.Sign(tt.claim, testRecordCID, tt.signer, tt.opts...)
			require.ErrorContains(t, err, tt.wantErr)
			requireUntouched(t, tt.claim)
		})
	}
}

func TestSign_ResigningReplacesStaleFields(t *testing.T) {
	key := testutil.NewKey(t, "ES256")
	signer := keySigner(t, key)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID}
	require.NoError(t, identity.Sign(claim, testRecordCID, signer, identity.WithCertificate(testutil.SelfSignedCertPEM(t, key, testSubjectSVID))))
	require.NotEmpty(t, claim.GetCertificate())

	// Signed again without a certificate, the old one must not linger.
	require.NoError(t, identity.Sign(claim, otherRecordCID, signer))
	require.Equal(t, otherRecordCID, claim.GetRecordCid())
	require.Nil(t, claim.Certificate)
}

func TestVerify_RejectsInvalidInput(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	signed := func(mutate func(*identityv1.Claim)) *identityv1.Claim {
		claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
		mutate(claim)
		require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))

		return claim
	}

	malformed := "next tuesday"

	tests := map[string]struct {
		claim    *identityv1.Claim
		expected string
		keys     []crypto.PublicKey
		wantErr  string
	}{
		"nil claim":           {nil, testSubjectDNS, []crypto.PublicKey{key.Public()}, "claim is nil"},
		"no declared subject": {signed(func(*identityv1.Claim) {}), "", []crypto.PublicKey{key.Public()}, "does not declare"},
		"no keys":             {signed(func(*identityv1.Claim) {}), testSubjectDNS, nil, "no public keys"},
		"malformed expiry":    {signed(func(c *identityv1.Claim) { c.ExpiresAt = &malformed }), testSubjectDNS, []crypto.PublicKey{key.Public()}, "invalid claim expiry"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ok, err := identity.Verify(tt.claim, testRecordCID, tt.expected, tt.keys...)
			require.False(t, ok)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCheck(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, identity.Sign(claim, testRecordCID, keySigner(t, key)))

	// It needs no key: a valid claim passes without any.
	require.NoError(t, identity.Check(claim, testRecordCID, testSubjectDNS))

	require.ErrorContains(t, identity.Check(nil, testRecordCID, testSubjectDNS), "claim is nil")
	require.ErrorContains(t, identity.Check(claim, otherRecordCID, testSubjectDNS), "record_cid")
	require.ErrorContains(t, identity.Check(claim, testRecordCID, testSubjectDID), "does not match")
	require.ErrorContains(t, identity.Check(claim, testRecordCID, ""), "does not declare")

	expired := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	claim.ExpiresAt = &expired
	require.ErrorContains(t, identity.Check(claim, testRecordCID, testSubjectDNS), "expired")

	// It does not look at the signature: a tampered claim still passes Check, and
	// only Verify rejects it.
	claim.ExpiresAt = nil
	claim.Signature = "tampered"
	require.NoError(t, identity.Check(claim, testRecordCID, testSubjectDNS))

	ok, err := identity.Verify(claim, testRecordCID, testSubjectDNS, key.Public())
	require.Error(t, err)
	require.False(t, ok)
}
