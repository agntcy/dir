// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package jws_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/agntcy/dir/client/utils/internal/testutil"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"
)

func TestToPublicKey(t *testing.T) {
	p256, ok := testutil.NewKey(t, "ES256").(*ecdsa.PrivateKey)
	require.True(t, ok)

	p521, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	require.NoError(t, err)

	rsa2048, ok := testutil.NewKey(t, "RS256").(*rsa.PrivateKey)
	require.True(t, ok)

	rsa1024 := testutil.NewSmallRSAKey(t)

	edPub, ok := testutil.NewKey(t, "EdDSA").Public().(ed25519.PublicKey)
	require.True(t, ok)

	// Supported keys come back usable, value types normalized to pointers.
	for name, key := range map[string]any{
		"ecdsa pointer": &p256.PublicKey,
		"ecdsa value":   p256.PublicKey,
		"rsa pointer":   &rsa2048.PublicKey,
		"rsa value":     rsa2048.PublicKey,
		"ed25519":       edPub,
	} {
		got, ok := jws.ToPublicKey(key)
		require.True(t, ok, name)

		_, isValue := got.(ecdsa.PublicKey)
		require.False(t, isValue, name)

		_, isValue = got.(rsa.PublicKey)
		require.False(t, isValue, name)
	}

	// Anything jws couldn't verify with is refused up front.
	for name, key := range map[string]any{
		"unsupported curve": &p521.PublicKey,
		"small rsa":         &rsa1024.PublicKey,
		"string":            "not a key",
		"symmetric":         []byte("secret"),
		"nil":               nil,
		// Malformed values must be refused, not crash key resolution.
		"typed nil ecdsa":   (*ecdsa.PublicKey)(nil),
		"typed nil rsa":     (*rsa.PublicKey)(nil),
		"ecdsa no curve":    &ecdsa.PublicKey{},
		"ecdsa value empty": ecdsa.PublicKey{},
		"rsa no modulus":    &rsa.PublicKey{},
		"rsa value empty":   rsa.PublicKey{},
	} {
		_, ok := jws.ToPublicKey(key)
		require.False(t, ok, name)
	}
}

func TestPublicKeyFromJWK(t *testing.T) {
	edPub, ok := testutil.NewKey(t, "EdDSA").Public().(ed25519.PublicKey)
	require.True(t, ok)

	newKey := func(use string) jwk.Key {
		key, err := jwk.FromRaw(edPub)
		require.NoError(t, err)

		if use != "" {
			require.NoError(t, key.Set(jwk.KeyUsageKey, use))
		}

		return key
	}

	for _, use := range []string{"", "sig"} {
		got, ok := jws.PublicKeyFromJWK(newKey(use))
		require.True(t, ok, "use=%q", use)
		require.Equal(t, edPub, got)
	}

	_, ok = jws.PublicKeyFromJWK(newKey("enc"))
	require.False(t, ok, "encryption keys are skipped")

	// key_ops, when present, must allow verification.
	withOps := func(ops ...jwk.KeyOperation) jwk.Key {
		key := newKey("")
		require.NoError(t, key.Set(jwk.KeyOpsKey, jwk.KeyOperationList(ops)))

		return key
	}

	for name, ops := range map[string][]jwk.KeyOperation{
		"verify":          {jwk.KeyOpVerify},
		"sign and verify": {jwk.KeyOpSign, jwk.KeyOpVerify},
	} {
		_, ok = jws.PublicKeyFromJWK(withOps(ops...))
		require.True(t, ok, name)
	}

	for name, ops := range map[string][]jwk.KeyOperation{
		"encrypt":     {jwk.KeyOpEncrypt},
		"sign only":   {jwk.KeyOpSign},
		"wrap/derive": {jwk.KeyOpWrapKey, jwk.KeyOpDeriveKey},
	} {
		_, ok = jws.PublicKeyFromJWK(withOps(ops...))
		require.False(t, ok, name)
	}

	// A private JWK yields its public half.
	priv, err := jwk.FromRaw(testutil.NewKey(t, "ES256"))
	require.NoError(t, err)

	got, ok := jws.PublicKeyFromJWK(priv)
	require.True(t, ok)
	require.IsType(t, &ecdsa.PublicKey{}, got)

	// Symmetric keys can't verify a signature.
	oct, err := jwk.FromRaw([]byte("secret"))
	require.NoError(t, err)

	_, ok = jws.PublicKeyFromJWK(oct)
	require.False(t, ok)
}
