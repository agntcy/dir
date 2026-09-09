// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package wellknown

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyAgainstKeySet_MatchesPublishedKey(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	jwkKey, err := jwk.FromRaw(priv.PublicKey)
	require.NoError(t, err)

	keySet := jwk.NewSet()
	require.NoError(t, keySet.AddKey(jwkKey))

	payload := []byte("hello world")

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	signer, err := identityv1.NewKeySigner(keyPEM)
	require.NoError(t, err)
	jwsCompact, err := signer.Sign(payload)
	require.NoError(t, err)

	assert.True(t, verifyAgainstKeySet(keySet, []byte(jwsCompact), payload))
}

func TestVerifyAgainstKeySet_NoMatch(t *testing.T) {
	keySet := jwk.NewSet()
	assert.False(t, verifyAgainstKeySet(keySet, []byte("sig"), []byte("payload")))
}

func TestDomainFromSubject(t *testing.T) {
	tests := []struct {
		subject string
		want    string
		wantErr bool
	}{
		{subject: "https://acme.com/agent", want: "acme.com"},
		{subject: "http://acme.com", want: "acme.com"},
		{subject: "acme.com", want: "acme.com"},
		{subject: "https://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			got, err := domainFromSubject(tt.subject)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
