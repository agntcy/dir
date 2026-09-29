// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package jws

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/youmark/pkcs8"
)

func generateKey(t *testing.T, kind string) crypto.Signer {
	t.Helper()

	switch kind {
	case "ES256":
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		return key
	case "ES384":
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		require.NoError(t, err)

		return key
	case "EdDSA":
		_, key, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		return key
	case "RS256":
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		return key
	default:
		t.Fatalf("unsupported key kind %q", kind)

		return nil
	}
}

func keyPEM(t *testing.T, key crypto.Signer) []byte {
	t.Helper()

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func selfSignedCertPEM(t *testing.T, key crypto.Signer, uriSAN string) []byte {
	t.Helper()

	uri, err := url.Parse(uriSAN)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: uriSAN},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestSignVerify_RoundTrip(t *testing.T) {
	payload := []byte("payload")

	for _, kind := range []string{"ES256", "ES384", "EdDSA", "RS256"} {
		t.Run(kind, func(t *testing.T) {
			key := generateKey(t, kind)

			sig, err := Sign(key, payload)
			require.NoError(t, err)
			require.NoError(t, Verify(sig, payload, key.Public()))
		})
	}
}

func TestVerify_Rejects(t *testing.T) {
	key := generateKey(t, "ES256")
	sig, err := Sign(key, []byte("payload"))
	require.NoError(t, err)

	require.Error(t, Verify(sig, []byte("tampered"), key.Public()), "tampered payload")
	require.Error(t, Verify(sig, []byte("payload"), generateKey(t, "ES256").Public()), "wrong key")
	require.Error(t, Verify(sig, []byte("payload"), generateKey(t, "EdDSA").Public()), "wrong key type")
	require.Error(t, Verify(sig, []byte("payload"), "not a key"), "unsupported key type")
}

func TestVerify_MultipleKeys(t *testing.T) {
	payload := []byte("payload")
	key := generateKey(t, "ES256")
	sig, err := Sign(key, payload)
	require.NoError(t, err)

	wrongES := generateKey(t, "ES256").Public()
	wrongEd := generateKey(t, "EdDSA").Public()

	require.NoError(t, Verify(sig, payload, wrongES, wrongEd, key.Public()), "matching key last")
	require.NoError(t, Verify(sig, payload, "not a key", key.Public()), "unusable key is skipped")
	require.Error(t, Verify(sig, payload, wrongES, wrongEd), "no key matches")
	require.Error(t, Verify(sig, payload), "no keys")
}

func TestKeySigner(t *testing.T) {
	for _, kind := range []string{"ES256", "EdDSA", "RS256"} {
		t.Run(kind, func(t *testing.T) {
			key := generateKey(t, kind)

			signer, err := NewKeySigner(keyPEM(t, key), nil)
			require.NoError(t, err)

			sig, err := signer.Sign([]byte("payload"))
			require.NoError(t, err)
			require.NoError(t, Verify(sig, []byte("payload"), key.Public()))
		})
	}

	_, err := NewKeySigner([]byte("not pem"), nil)
	require.Error(t, err)
}

func TestKeyCertSigner(t *testing.T) {
	const spiffeID = "spiffe://acme.com/agents/finance"

	key := generateKey(t, "ES256")

	signer, err := NewKeyCertSigner(keyPEM(t, key), selfSignedCertPEM(t, key, spiffeID), nil)
	require.NoError(t, err)
	require.NotEmpty(t, signer.CertificateDER())
	require.True(t, signer.SubjectMatchesCertificate(spiffeID))
	require.False(t, signer.SubjectMatchesCertificate("spiffe://acme.com/agents/other"))

	sig, err := signer.Sign([]byte("payload"))
	require.NoError(t, err)
	require.NoError(t, Verify(sig, []byte("payload"), key.Public()))
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
			key := generateKey(t, kind)
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
	key := generateKey(t, "ES256")
	encrypted := encryptedKeyPEM(t, key, "s3cret")

	_, err := NewKeySigner(encrypted, nil)
	require.ErrorContains(t, err, "password is required")

	_, err = NewKeySigner(encrypted, []byte("wrong"))
	require.ErrorContains(t, err, "decrypt private key")
}

func TestKeySigner_PasswordIgnoredForUnencryptedKey(t *testing.T) {
	key := generateKey(t, "ES256")

	_, err := NewKeySigner(keyPEM(t, key), []byte("unused"))
	require.NoError(t, err)
}

func TestKeyCertSigner_EncryptedKey(t *testing.T) {
	const spiffeID = "spiffe://acme.com/agents/finance"

	key := generateKey(t, "ES256")

	signer, err := NewKeyCertSigner(encryptedKeyPEM(t, key, "s3cret"), selfSignedCertPEM(t, key, spiffeID), []byte("s3cret"))
	require.NoError(t, err)
	require.True(t, signer.SubjectMatchesCertificate(spiffeID))
}

func TestRSAMinimumKeySize(t *testing.T) {
	small, err := rsa.GenerateKey(rand.Reader, 1024) //nolint:gosec // deliberately below the minimum to test the size floor
	require.NoError(t, err)

	_, err = Sign(small, []byte("payload"))
	require.ErrorContains(t, err, "too small")

	require.ErrorContains(t, Verify("a.b.c", []byte("payload"), small.Public()), "too small")
}
