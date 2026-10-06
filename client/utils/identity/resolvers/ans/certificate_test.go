// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCertificate(t *testing.T) {
	p521Key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	require.NoError(t, err)

	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	valid := func(ansName string) identityCert {
		return mintIdentityCert(t, ansName, testNow.Add(-time.Hour), testNow.Add(time.Hour))
	}

	tests := []struct {
		name    string
		cert    identityCert
		now     time.Time
		wantErr string
	}{
		{name: "names the agent", cert: valid(testAnsName), now: testNow},
		{name: "second san names the agent", cert: mintIdentityCertWithSANs(t, "spiffe://trust.example.com/agent", testAnsName), now: testNow},
		{name: "ed25519 key", cert: mintIdentityCertWithKey(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour), edKey), now: testNow},
		{name: "valid at not before", cert: mintIdentityCert(t, testAnsName, testNow, testNow.Add(time.Hour)), now: testNow},
		{name: "valid at not after", cert: mintIdentityCert(t, testAnsName, testNow.Add(-time.Hour), testNow), now: testNow},
		{name: "uppercase san is another spelling", cert: valid("ans://v1.0.0.AGENT.example.com"), now: testNow, wantErr: "ans certificate: certificate does not name"},
		{name: "other host", cert: valid("ans://v1.0.0.other.example.com"), now: testNow, wantErr: `ans certificate: certificate does not name "ans://v1.0.0.agent.example.com" as a URI SAN`},
		{name: "other version", cert: valid("ans://v1.0.1.agent.example.com"), now: testNow, wantErr: "does not name"},
		{name: "no ans san", cert: valid("spiffe://trust.example.com/agent"), now: testNow, wantErr: "does not name"},
		{name: "no san at all", cert: mintIdentityCertWithSANs(t), now: testNow, wantErr: "does not name"},
		{name: "expired", cert: valid(testAnsName), now: testNow.Add(2 * time.Hour), wantErr: "ans certificate: certificate is not valid at 2026-10-06T14:00:00Z"},
		{name: "not yet valid", cert: valid(testAnsName), now: testNow.Add(-2 * time.Hour), wantErr: "is not valid at"},
		{name: "p-521 key", cert: mintIdentityCertWithKey(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour), p521Key), now: testNow, wantErr: "ans certificate: unsupported public key type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := checkCertificate(tt.cert.cert, testAnsName, tt.now)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, key)

				return
			}

			require.NoError(t, err)

			equaler, ok := key.(interface{ Equal(crypto.PublicKey) bool })
			require.True(t, ok)
			assert.True(t, equaler.Equal(tt.cert.key.Public()), "returned key is not the certificate's")
		})
	}
}
