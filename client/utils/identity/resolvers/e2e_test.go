// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"crypto"
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"testing"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/stretchr/testify/require"
)

const e2eRecordCID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

func keySigner(t *testing.T, key crypto.PrivateKey) *jws.KeySigner {
	t.Helper()

	signer, err := jws.NewKeySigner(pkcs8PEM(t, key), nil)
	require.NoError(t, err)

	return signer
}

// signedClaim signs a fresh claim for subject the way a real publisher would.
func signedClaim(t *testing.T, subject string, key crypto.PrivateKey) *identityv1.Claim {
	t.Helper()

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: subject}
	require.NoError(t, identity.Sign(claim, e2eRecordCID, keySigner(t, key)))

	return claim
}

// verifyWith is the flow a real caller follows: resolve the subject's keys,
// then verify the claim against all of them.
func verifyWith(t *testing.T, r Resolver, claim *identityv1.Claim) (bool, error) {
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

	ok, err := identity.Verify(claim, e2eRecordCID, claim.GetSubject(), keys...)
	if err != nil {
		return false, fmt.Errorf("verify: %w", err)
	}

	return ok, nil
}

// The resolvers are only useful if what they return is accepted by the real
// signing code, so each scheme is exercised sign -> resolve -> verify.
func TestEndToEnd_DNS(t *testing.T) {
	key := newECKey(t, elliptic.P256())

	resolver := &DNS{lookupTXT: fakeTXT(map[string][]string{
		"_agntcy-key.acme.com": {"v=akv1;key=" + spkiBase64(t, &key.PublicKey)},
	}, nil)}

	ok, err := verifyWith(t, resolver, signedClaim(t, "dns:acme.com", key))
	require.NoError(t, err)
	require.True(t, ok)

	// A claim signed by someone who doesn't hold the published key is rejected.
	ok, err = verifyWith(t, resolver, signedClaim(t, "dns:acme.com", newECKey(t, elliptic.P256())))
	require.Error(t, err)
	require.False(t, ok)
}

func TestEndToEnd_WellKnown(t *testing.T) {
	key := newEdKey(t)

	resolver := NewWellKnown(&fakeFetcher{docs: map[string][]byte{
		"https://acme.com/.well-known/jwks.json": jwksJSON(t, key.Public()),
	}})

	ok, err := verifyWith(t, resolver, signedClaim(t, "https://acme.com/agents/finance", key))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEndToEnd_DIDWeb(t *testing.T) {
	key := newRSAKey(t)

	resolver := NewDID(&fakeFetcher{docs: map[string][]byte{
		"https://acme.com/.well-known/did.json": didDoc(t, "did:web:acme.com", &key.PublicKey),
	}})

	ok, err := verifyWith(t, resolver, signedClaim(t, "did:web:acme.com", key))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEndToEnd_DIDKey(t *testing.T) {
	key := newEdKey(t)
	subject := "did:key:" + encodeDIDKey(t, multicodecEd25519Pub, edPublic(t, key))

	ok, err := verifyWith(t, NewDID(&fakeFetcher{}), signedClaim(t, subject, key))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEndToEnd_SPIFFE(t *testing.T) {
	ca := newTestCA(t, "acme root")
	key := newECKey(t, elliptic.P256())

	certSigner, err := jws.NewKeyCertSigner(pkcs8PEM(t, key), pemCertificate(ca.issue(t, key, svidOpts{uris: []string{svidSubject}})), nil)
	require.NoError(t, err)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: svidSubject}
	require.NoError(t, identity.Sign(claim, e2eRecordCID, certSigner,
		identity.WithSubjectChecker(certSigner.SubjectMatchesCertificate),
		identity.WithCertificateProvider(certSigner.CertificateDER)))

	ok, err := verifyWith(t, NewSPIFFE(bundleFor(t, "acme.com", ca)), claim)
	require.NoError(t, err)
	require.True(t, ok)

	// With no trust bundle for the subject's domain, the same claim fails closed.
	ok, err = verifyWith(t, NewSPIFFE(bundleFor(t, "unrelated.org", ca)), claim)
	require.Error(t, err)
	require.False(t, ok)
}
