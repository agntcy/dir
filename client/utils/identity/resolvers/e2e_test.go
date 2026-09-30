// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers_test

import (
	"context"
	"crypto"
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"testing"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	didresolver "github.com/agntcy/dir/client/utils/identity/resolvers/did"
	dnsresolver "github.com/agntcy/dir/client/utils/identity/resolvers/dns"
	"github.com/agntcy/dir/client/utils/identity/resolvers/internal/testutil"
	spifferesolver "github.com/agntcy/dir/client/utils/identity/resolvers/spiffe"
	wellknownresolver "github.com/agntcy/dir/client/utils/identity/resolvers/wellknown"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/stretchr/testify/require"
)

const (
	recordCID   = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
	svidSubject = "spiffe://acme.com/agents/finance"

	multicodecEd25519Pub = 0xed
)

// Every resolver must satisfy the shared contract.
var (
	_ resolvers.Resolver = (*dnsresolver.Resolver)(nil)
	_ resolvers.Resolver = (*didresolver.Resolver)(nil)
	_ resolvers.Resolver = (*wellknownresolver.Resolver)(nil)
	_ resolvers.Resolver = (*spifferesolver.Resolver)(nil)
)

// signedClaim signs a fresh claim for subject the way a real publisher would.
func signedClaim(t *testing.T, subject string, key crypto.PrivateKey) *identityv1.Claim {
	t.Helper()

	signer, err := jws.NewKeySigner(testutil.PKCS8PEM(t, key), nil)
	require.NoError(t, err)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: subject}
	require.NoError(t, identity.Sign(claim, recordCID, signer))

	return claim
}

// verifyWith is the flow a real caller follows: resolve the subject's keys,
// then verify the claim against all of them.
func verifyWith(t *testing.T, r resolvers.Resolver, claim *identityv1.Claim) (bool, error) {
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

	ok, err := identity.Verify(claim, recordCID, claim.GetSubject(), keys...)
	if err != nil {
		return false, fmt.Errorf("verify: %w", err)
	}

	return ok, nil
}

// The resolvers are only useful if what they return is accepted by the real
// signing code, so each scheme is exercised sign -> resolve -> verify.
func TestEndToEnd_DNS(t *testing.T) {
	key := testutil.NewECKey(t, elliptic.P256())

	resolver := dnsresolver.New(dnsresolver.WithLookupTXT(func(_ context.Context, name string) ([]string, error) {
		require.Equal(t, "_agntcy-key.acme.com", name)

		return []string{"v=akv1;key=" + testutil.SPKIBase64(t, &key.PublicKey)}, nil
	}))

	ok, err := verifyWith(t, resolver, signedClaim(t, "dns:acme.com", key))
	require.NoError(t, err)
	require.True(t, ok)

	// A claim signed by someone who doesn't hold the published key is rejected.
	ok, err = verifyWith(t, resolver, signedClaim(t, "dns:acme.com", testutil.NewECKey(t, elliptic.P256())))
	require.Error(t, err)
	require.False(t, ok)
}

func TestEndToEnd_WellKnown(t *testing.T) {
	key := testutil.NewEdKey(t)

	resolver := wellknownresolver.New(&testutil.FakeFetcher{Docs: map[string][]byte{
		"https://acme.com/.well-known/jwks.json": testutil.JWKS(t, key.Public()),
	}})

	ok, err := verifyWith(t, resolver, signedClaim(t, "https://acme.com/agents/finance", key))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEndToEnd_DIDWeb(t *testing.T) {
	key := testutil.NewRSAKey(t)

	resolver := didresolver.New(&testutil.FakeFetcher{Docs: map[string][]byte{
		"https://acme.com/.well-known/did.json": testutil.DIDDocument(t, "did:web:acme.com", &key.PublicKey),
	}})

	ok, err := verifyWith(t, resolver, signedClaim(t, "did:web:acme.com", key))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEndToEnd_DIDKey(t *testing.T) {
	key := testutil.NewEdKey(t)
	subject := "did:key:" + testutil.EncodeDIDKey(t, multicodecEd25519Pub, testutil.EdPublic(t, key))

	ok, err := verifyWith(t, didresolver.New(&testutil.FakeFetcher{}), signedClaim(t, subject, key))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEndToEnd_SPIFFE(t *testing.T) {
	ca := testutil.NewCA(t, "acme root")
	key := testutil.NewECKey(t, elliptic.P256())

	svid := testutil.CertificatePEM(ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}}))

	certSigner, err := jws.NewKeyCertSigner(testutil.PKCS8PEM(t, key), svid, nil)
	require.NoError(t, err)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: svidSubject}
	require.NoError(t, identity.Sign(claim, recordCID, certSigner,
		identity.WithSubjectChecker(certSigner.SubjectMatchesCertificate),
		identity.WithCertificateProvider(certSigner.CertificateDER)))

	ok, err := verifyWith(t, spifferesolver.New(testutil.Bundle(t, "acme.com", ca)), claim)
	require.NoError(t, err)
	require.True(t, ok)

	// With no trust bundle for the subject's domain, the same claim fails closed.
	ok, err = verifyWith(t, spifferesolver.New(testutil.Bundle(t, "unrelated.org", ca)), claim)
	require.Error(t, err)
	require.False(t, ok)
}
