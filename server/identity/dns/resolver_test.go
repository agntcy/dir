// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolver_Verify_MatchesPublishedKey(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	spkiDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)

	record := "v=akv1;key=" + base64.StdEncoding.EncodeToString(spkiDER)

	payload := []byte("hello world")

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	signer, err := identityv1.NewKeySigner(keyPEM)
	require.NoError(t, err)
	jwsCompact, err := signer.Sign(payload)
	require.NoError(t, err)

	resolver := &Resolver{
		lookupTXT: func(_ context.Context, name string) ([]string, error) {
			assert.Equal(t, TXTPrefix+"acme.com", name)

			return []string{record}, nil
		},
	}

	ok, err := resolver.Verify(t.Context(), "dns:acme.com", []byte(jwsCompact), payload)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestResolver_Verify_NoMatchingKey(t *testing.T) {
	resolver := &Resolver{
		lookupTXT: func(_ context.Context, _ string) ([]string, error) {
			return []string{"v=akv1;key=" + base64.StdEncoding.EncodeToString([]byte("not-a-key"))}, nil
		},
	}

	ok, err := resolver.Verify(t.Context(), "dns:acme.com", []byte("sig"), []byte("payload"))
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestParseKeyRecord(t *testing.T) {
	key, ok := parseKeyRecord("v=akv1;key=aGVsbG8=")
	assert.True(t, ok)
	assert.Equal(t, []byte("hello"), key)

	_, ok = parseKeyRecord("v=other;key=aGVsbG8=")
	assert.False(t, ok)

	_, ok = parseKeyRecord("v=akv1")
	assert.False(t, ok)
}
