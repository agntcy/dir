// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package spiffe_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agntcy/dir/server/identity/spiffe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestCACert(t *testing.T) string {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))

	return path
}

func TestBundles_TrustedCerts(t *testing.T) {
	caPath := writeTestCACert(t)

	bundles, err := spiffe.Load(spiffe.Config{TrustDomains: map[string]string{"acme.com": caPath}})
	require.NoError(t, err)

	certs := bundles.TrustedCerts("spiffe://acme.com/agents/finance")
	assert.Len(t, certs, 1)

	assert.Nil(t, bundles.TrustedCerts("spiffe://other.org/agent"), "unconfigured trust domain returns nil")
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := spiffe.Load(spiffe.Config{TrustDomains: map[string]string{"acme.com": "/nonexistent/ca.pem"}})
	require.Error(t, err)
}
