// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKeyPassword = "test"

func generateKeyPair(t *testing.T, keyDetails v1.PublicKeyDetails) *cosign.KeysBytes {
	t.Helper()

	algo, err := signature.GetAlgorithmDetails(keyDetails)
	require.NoError(t, err)

	keys, err := cosign.GenerateKeyPairWithAlgorithm(&algo, func(bool) ([]byte, error) {
		return []byte(testKeyPassword), nil
	})
	require.NoError(t, err)

	return keys
}

func TestSignVerifyWithKeyRoundTrip(t *testing.T) {
	payload := []byte("bafytestcid")

	tests := []struct {
		name       string
		keyDetails v1.PublicKeyDetails
		algorithm  string
	}{
		{name: "ECDSA P-256", keyDetails: v1.PublicKeyDetails_PKIX_ECDSA_P256_SHA_256, algorithm: "ECDSA-P-256"},
		{name: "Ed25519", keyDetails: v1.PublicKeyDetails_PKIX_ED25519, algorithm: "Ed25519"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := generateKeyPair(t, tt.keyDetails)

			sig, pubKey, err := SignBlobWithKey(t.Context(), payload, &signv1.SignWithKey{
				PrivateKey: string(keys.PrivateBytes),
				Password:   []byte(testKeyPassword),
			})
			require.NoError(t, err)
			assert.Equal(t, tt.algorithm, sig.GetAlgorithm())

			info, err := VerifyWithKeys(t.Context(), payload, []string{string(keys.PublicBytes)}, sig)
			require.NoError(t, err)
			assert.Equal(t, pubKey.GetKey(), info.GetKey().GetPublicKey())
			assert.Equal(t, tt.algorithm, info.GetKey().GetAlgorithm())

			otherKeys := generateKeyPair(t, tt.keyDetails)
			_, err = VerifyWithKeys(t.Context(), payload, []string{string(otherKeys.PublicBytes)}, sig)
			require.Error(t, err)

			_, err = VerifyWithKeys(t.Context(), []byte("other-payload"), []string{string(keys.PublicBytes)}, sig)
			require.Error(t, err)
		})
	}
}

func TestVerifyWithKeysAcceptsPureEd25519(t *testing.T) {
	payload := []byte("bafytestcid")

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pubKeyPEM, err := cryptoutils.MarshalPublicKeyToPEM(pub)
	require.NoError(t, err)

	sig := &signv1.Signature{
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload)),
	}

	info, err := VerifyWithKeys(t.Context(), payload, []string{string(pubKeyPEM)}, sig)
	require.NoError(t, err)
	assert.Equal(t, "Ed25519", info.GetKey().GetAlgorithm())
}
