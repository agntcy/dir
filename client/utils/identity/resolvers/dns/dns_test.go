// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dnsresolver

import (
	"context"
	"crypto"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/internal/testutil"
	"github.com/stretchr/testify/require"
)

func fakeTXT(records map[string][]string, queried *[]string) func(context.Context, string) ([]string, error) {
	return func(_ context.Context, name string) ([]string, error) {
		if queried != nil {
			*queried = append(*queried, name)
		}

		txt, ok := records[name]
		if !ok {
			return nil, errors.New("no such host")
		}

		return txt, nil
	}
}

func TestDNS_Resolve(t *testing.T) {
	first := testutil.NewECKey(t, elliptic.P256())
	second := testutil.NewEdKey(t)

	var queried []string

	resolver := New(WithLookupTXT(fakeTXT(map[string][]string{
		"_agntcy-key.acme.com": {
			"v=akv1;key=" + spkiBase64(t, &first.PublicKey),
			"v=spf1 include:_spf.google.com ~all", // unrelated record
			"v=akv1;key=" + spkiBase64(t, second.Public()),
			"v=akv2;key=" + spkiBase64(t, &first.PublicKey), // wrong version
			"v=akv1;key=not-base64!",
			"v=akv1;key=", // empty key
		},
	}, &queried)))

	for _, subject := range []string{"dns:acme.com", "acme.com"} {
		keys, err := resolver.Resolve(t.Context(), subject, nil)
		require.NoError(t, err, subject)
		testutil.RequireSameKeys(t, []crypto.PublicKey{&first.PublicKey, second.Public()}, keys)
	}

	require.Equal(t, []string{"_agntcy-key.acme.com", "_agntcy-key.acme.com"}, queried)
}

func TestDNS_Resolve_Errors(t *testing.T) {
	resolver := New(WithLookupTXT(fakeTXT(map[string][]string{
		"_agntcy-key.empty.com":   {},
		"_agntcy-key.garbage.com": {"hello", "v=akv1", "key=abc"},
	}, nil)))

	_, err := resolver.Resolve(t.Context(), "dns:empty.com", nil)
	require.ErrorIs(t, err, resolvers.ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "dns:garbage.com", nil)
	require.ErrorIs(t, err, resolvers.ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "dns:missing.com", nil)
	require.ErrorContains(t, err, "lookup TXT records for _agntcy-key.missing.com")

	for _, subject := range []string{"dns:", ""} {
		_, err = resolver.Resolve(t.Context(), subject, nil)
		require.ErrorContains(t, err, "invalid dns subject", subject)
	}
}

// spkiBase64 encodes pub as base64 DER SubjectPublicKeyInfo, as published in a TXT record.
func spkiBase64(t *testing.T, pub crypto.PublicKey) string {
	t.Helper()

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(der)
}

func TestResolver_SatisfiesContract(t *testing.T) {
	var _ resolvers.Resolver = New()
}

func TestEndToEnd(t *testing.T) {
	key := testutil.NewECKey(t, elliptic.P256())

	resolver := New(WithLookupTXT(func(_ context.Context, name string) ([]string, error) {
		require.Equal(t, "_agntcy-key.acme.com", name)

		return []string{"v=akv1;key=" + spkiBase64(t, &key.PublicKey)}, nil
	}))

	ok, err := testutil.VerifyClaim(t, resolver, testutil.SignedClaim(t, "dns:acme.com", key))
	require.NoError(t, err)
	require.True(t, ok)

	// A claim signed by someone who doesn't hold the published key is rejected.
	imposter := testutil.NewECKey(t, elliptic.P256())

	ok, err = testutil.VerifyClaim(t, resolver, testutil.SignedClaim(t, "dns:acme.com", imposter))
	require.Error(t, err)
	require.False(t, ok)
}
