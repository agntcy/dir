// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupJWTTLSAuth_Validation(t *testing.T) {
	t.Parallel()

	t.Run("missing socket path", func(t *testing.T) {
		t.Parallel()

		opts := &options{
			config: &Config{
				ServerAddress: testServerAddr,
				AuthMode:      "jwt-tls",
				JWTAudience:   testJWTAudience,
			},
		}

		err := opts.setupJWTTLSAuth(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "spiffe socket path is required")
	})

	t.Run("missing audience", func(t *testing.T) {
		t.Parallel()

		opts := &options{
			config: &Config{
				ServerAddress:    testServerAddr,
				AuthMode:         "jwt-tls",
				SpiffeSocketPath: testSpiffeSocket,
			},
		}

		err := opts.setupJWTTLSAuth(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT audience is required")
	})
}

func TestBuildWebPKITLSConfig(t *testing.T) {
	t.Parallel()

	t.Run("uses server name from address", func(t *testing.T) {
		t.Parallel()

		opts := &options{
			config: &Config{
				ServerAddress: "ads.outshift.io:443",
			},
		}

		tlsCfg, err := opts.buildWebPKITLSConfig()
		require.NoError(t, err)
		assert.Equal(t, "ads.outshift.io", tlsCfg.ServerName)
		assert.Equal(t, uint16(tls.VersionTLS12), tlsCfg.MinVersion)
		assert.Nil(t, tlsCfg.RootCAs)
	})

	t.Run("loads optional custom CA", func(t *testing.T) {
		t.Parallel()

		caPath := writeTestRootCA(t)

		opts := &options{
			config: &Config{
				ServerAddress: "ads.outshift.io:443",
				TlsCAFile:     caPath,
			},
		}

		tlsCfg, err := opts.buildWebPKITLSConfig()
		require.NoError(t, err)
		require.NotNil(t, tlsCfg.RootCAs)
	})

	t.Run("invalid CA pem", func(t *testing.T) {
		t.Parallel()

		caPath := filepath.Join(t.TempDir(), "ca.pem")
		require.NoError(t, os.WriteFile(caPath, []byte("not-a-cert"), 0o600))

		opts := &options{
			config: &Config{
				ServerAddress: "ads.outshift.io:443",
				TlsCAFile:     caPath,
			},
		}

		_, err := opts.buildWebPKITLSConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to append root CA")
	})

	t.Run("invalid CA file", func(t *testing.T) {
		t.Parallel()

		opts := &options{
			config: &Config{
				ServerAddress: "ads.outshift.io:443",
				TlsCAFile:     filepath.Join(t.TempDir(), "missing.pem"),
			},
		}

		_, err := opts.buildWebPKITLSConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read TLS CA file")
	})
}

func TestWithAuth_JWTTLSSupported(t *testing.T) {
	t.Parallel()

	opts := &options{
		config: &Config{
			ServerAddress:    testServerAddr,
			AuthMode:         "jwt-tls",
			SpiffeSocketPath: "/tmp/nonexistent-jwt-tls.sock",
			JWTAudience:      testJWTAudience,
		},
	}

	err := withAuth(context.Background())(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create SPIFFE client")
	assert.NotContains(t, err.Error(), "unsupported auth mode")
}

func writeTestRootCA(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-root-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	}), 0o600))

	return caPath
}
