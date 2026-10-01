// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package jws

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/agntcy/dir/client/utils/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/youmark/pkcs8"
)

func TestSignVerify_RoundTrip(t *testing.T) {
	payload := []byte("payload")

	for _, kind := range []string{"ES256", "ES384", "EdDSA", "RS256"} {
		t.Run(kind, func(t *testing.T) {
			key := testutil.NewKey(t, kind)

			sig, err := Sign(key, payload)
			require.NoError(t, err)
			require.NoError(t, Verify(sig, payload, key.Public()))
		})
	}
}

func TestVerify_Rejects(t *testing.T) {
	key := testutil.NewKey(t, "ES256")
	sig, err := Sign(key, []byte("payload"))
	require.NoError(t, err)

	require.Error(t, Verify(sig, []byte("tampered"), key.Public()), "tampered payload")
	require.Error(t, Verify(sig, []byte("payload"), testutil.NewKey(t, "ES256").Public()), "wrong key")
	require.Error(t, Verify(sig, []byte("payload"), testutil.NewKey(t, "EdDSA").Public()), "wrong key type")
	require.Error(t, Verify(sig, []byte("payload"), "not a key"), "unsupported key type")
}

func TestVerify_MultipleKeys(t *testing.T) {
	payload := []byte("payload")
	key := testutil.NewKey(t, "ES256")
	sig, err := Sign(key, payload)
	require.NoError(t, err)

	wrongES := testutil.NewKey(t, "ES256").Public()
	wrongEd := testutil.NewKey(t, "EdDSA").Public()

	require.NoError(t, Verify(sig, payload, wrongES, wrongEd, key.Public()), "matching key last")
	require.NoError(t, Verify(sig, payload, "not a key", key.Public()), "unusable key is skipped")
	require.Error(t, Verify(sig, payload, wrongES, wrongEd), "no key matches")
	require.Error(t, Verify(sig, payload), "no keys")
}

func TestKeySigner(t *testing.T) {
	for _, kind := range []string{"ES256", "EdDSA", "RS256"} {
		t.Run(kind, func(t *testing.T) {
			key := testutil.NewKey(t, kind)

			signer, err := NewKeySigner(testutil.PKCS8PEM(t, key), nil)
			require.NoError(t, err)

			require.Equal(t, key.Public(), signer.Public())

			sig, err := signer.Sign([]byte("payload"))
			require.NoError(t, err)
			require.NoError(t, Verify(sig, []byte("payload"), key.Public()))
		})
	}

	_, err := NewKeySigner([]byte("not pem"), nil)
	require.Error(t, err)
}

func encryptedKeyPEM(t *testing.T, key crypto.Signer, password string) []byte {
	t.Helper()

	der, err := pkcs8.MarshalPrivateKey(key, []byte(password), pkcs8.DefaultOpts)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: der})
}

func TestKeySigner_EncryptedKey(t *testing.T) {
	for _, kind := range []string{"ES256", "EdDSA", "RS256"} {
		t.Run(kind, func(t *testing.T) {
			key := testutil.NewKey(t, kind)
			encrypted := encryptedKeyPEM(t, key, "s3cret")

			signer, err := NewKeySigner(encrypted, []byte("s3cret"))
			require.NoError(t, err)

			sig, err := signer.Sign([]byte("payload"))
			require.NoError(t, err)
			require.NoError(t, Verify(sig, []byte("payload"), key.Public()))
		})
	}
}

func TestKeySigner_EncryptedKeyErrors(t *testing.T) {
	key := testutil.NewKey(t, "ES256")
	encrypted := encryptedKeyPEM(t, key, "s3cret")

	_, err := NewKeySigner(encrypted, nil)
	require.ErrorContains(t, err, "password is required")

	_, err = NewKeySigner(encrypted, []byte("wrong"))
	require.ErrorContains(t, err, "decrypt private key")
}

func TestKeySigner_PasswordIgnoredForUnencryptedKey(t *testing.T) {
	key := testutil.NewKey(t, "ES256")

	_, err := NewKeySigner(testutil.PKCS8PEM(t, key), []byte("unused"))
	require.NoError(t, err)
}

func TestRSAMinimumKeySize(t *testing.T) {
	small := testutil.NewSmallRSAKey(t)

	_, err := Sign(small, []byte("payload"))
	require.ErrorContains(t, err, "too small")

	require.ErrorContains(t, Verify("a.b.c", []byte("payload"), small.Public()), "too small")
}

func TestKeySigner_LegacyPEMFormats(t *testing.T) {
	ec, ok := testutil.NewKey(t, "ES256").(*ecdsa.PrivateKey)
	require.True(t, ok)

	ecDER, err := x509.MarshalECPrivateKey(ec)
	require.NoError(t, err)

	rsaKey, ok := testutil.NewKey(t, "RS256").(*rsa.PrivateKey)
	require.True(t, ok)

	for name, tc := range map[string]struct {
		pem  []byte
		pub  crypto.PublicKey
		sign crypto.Signer
	}{
		"EC PRIVATE KEY":  {pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecDER}), ec.Public(), ec},
		"RSA PRIVATE KEY": {pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)}), rsaKey.Public(), rsaKey},
	} {
		t.Run(name, func(t *testing.T) {
			signer, err := NewKeySigner(tc.pem, nil)
			require.NoError(t, err)

			sig, err := signer.Sign([]byte("payload"))
			require.NoError(t, err)
			require.NoError(t, Verify(sig, []byte("payload"), tc.pub))
		})
	}
}

func TestKeySigner_RejectsBadKeyMaterial(t *testing.T) {
	garbage := func(blockType string) []byte {
		return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: []byte("garbage")})
	}

	for name, in := range map[string][]byte{
		"not pem":          []byte("nope"),
		"unsupported type": garbage("CERTIFICATE"),
		"bad EC":           garbage("EC PRIVATE KEY"),
		"bad RSA":          garbage("RSA PRIVATE KEY"),
		"bad PKCS8":        garbage("PRIVATE KEY"),
	} {
		_, err := NewKeySigner(in, nil)
		require.Error(t, err, name)
	}
}
