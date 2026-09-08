// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package did

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"testing"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/multiformats/go-multibase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDidWebURL(t *testing.T) {
	tests := []struct {
		subject string
		want    string
		wantErr bool
	}{
		{subject: "did:web:acme.com", want: "https://acme.com/.well-known/did.json"},
		{subject: "did:web:acme.com:agents:finance", want: "https://acme.com/agents/finance/did.json"},
		{subject: "did:web:", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			got, err := didWebURL(tt.subject)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// signWithKeySigner returns a real detached-payload JWS over payload,
// signed by priv, for use as test fixtures.
func signWithKeySigner(t *testing.T, priv *ecdsa.PrivateKey, payload []byte) string {
	t.Helper()

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	signer, err := identityv1.NewKeySigner(keyPEM)
	require.NoError(t, err)

	jwsCompact, err := signer.Sign(payload)
	require.NoError(t, err)

	return jwsCompact
}

func TestVerifyDIDDocument_MatchesPublishedKey(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	jwkKey, err := jwk.FromRaw(priv.PublicKey)
	require.NoError(t, err)

	jwkJSON, err := json.Marshal(jwkKey)
	require.NoError(t, err)

	doc := fmt.Sprintf(`{"verificationMethod":[{"id":"did:web:acme.com#key-1","publicKeyJwk":%s}]}`, jwkJSON)

	payload := []byte("hello world")
	jwsCompact := signWithKeySigner(t, priv, payload)

	ok, err := verifyDIDDocument([]byte(doc), []byte(jwsCompact), payload)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyDIDDocument_NoMatch(t *testing.T) {
	ok, err := verifyDIDDocument([]byte(`{"verificationMethod":[]}`), []byte("sig"), []byte("payload"))
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestVerifyDIDKey_P256_RoundTrip(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	compressed := elliptic.MarshalCompressed(elliptic.P256(), priv.PublicKey.X, priv.PublicKey.Y)

	prefixed := make([]byte, 0, len(compressed)+2)
	prefixed = binary.AppendUvarint(prefixed, multicodecP256Pub)
	prefixed = append(prefixed, compressed...)

	encoded, err := multibase.Encode(multibase.Base58BTC, prefixed)
	require.NoError(t, err)

	subject := "did:key:" + encoded

	payload := []byte("hello world")
	jwsCompact := signWithKeySigner(t, priv, payload)

	ok, err := verifyDIDKey(subject, []byte(jwsCompact), payload)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyDIDKey_WrongSignatureFails(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	compressed := elliptic.MarshalCompressed(elliptic.P256(), priv.PublicKey.X, priv.PublicKey.Y)
	prefixed := binary.AppendUvarint([]byte{}, multicodecP256Pub)
	prefixed = append(prefixed, compressed...)

	encoded, err := multibase.Encode(multibase.Base58BTC, prefixed)
	require.NoError(t, err)

	ok, err := verifyDIDKey("did:key:"+encoded, []byte("not-a-valid-signature"), []byte("payload"))
	require.NoError(t, err)
	assert.False(t, ok)
}
