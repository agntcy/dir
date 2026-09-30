// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSpiffeID        = "spiffe://acme.com/agents/finance"
	certificateLifetime = 48 * time.Hour
)

func TestCommand_HasExpectedSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command.Commands() {
		names[c.Name()] = true
	}

	assert.True(t, names["claim"])
	assert.True(t, names["status"])
	assert.True(t, names["resolve"])
}

func TestClaimCommand_FlagValidation(t *testing.T) {
	base := []string{"--record", "baguqeera-cid", "--role", "identity", "--subject", "did:web:acme.com"}

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "key alone", args: append(base, "--key", "key.pem")},
		{name: "key and cert", args: append(base, "--key", "key.pem", "--cert", "cert.pem")},
		{name: "cert without key", args: append(base, "--cert", "cert.pem"), wantErr: `"key" not set`},
		{name: "no key at all", args: base, wantErr: `"key" not set`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t, claimCmd.Flags())

			require.NoError(t, claimCmd.ParseFlags(tt.args))

			err := claimCmd.ValidateRequiredFlags()
			if err == nil {
				err = claimCmd.ValidateFlagGroups()
			}

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

// pflag keeps a flag's Changed state across ParseFlags calls on the same set.
func resetFlags(t *testing.T, flags *pflag.FlagSet) {
	t.Helper()

	flags.VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		require.NoError(t, f.Value.Set(f.DefValue))
	})
}

// mintKeyCertFiles writes a P-256 key and a self-signed certificate over it,
// carrying uri as its URI SAN and expiring at notAfter, into a temporary
// directory and returns their paths.
func mintKeyCertFiles(t *testing.T, uri string, notAfter time.Time) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	parsed, err := url.Parse(uri)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     notAfter,
		URIs:         []*url.URL{parsed},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.pem")
	certPath := filepath.Join(dir, "cert.pem")

	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))

	return keyPath, certPath
}

func TestLoadSigner(t *testing.T) {
	keyPath, certPath := mintKeyCertFiles(t, testSpiffeID, time.Now().Add(time.Hour))

	tests := []struct {
		name     string
		keyPath  string
		certPath string
		wantType any
		wantErr  string
	}{
		{name: "key alone builds a key signer", keyPath: keyPath, wantType: &identityv1.KeySigner{}},
		{name: "key and certificate build a spiffe signer", keyPath: keyPath, certPath: certPath, wantType: &identityv1.SpiffeSigner{}},
		{name: "key is required", wantErr: "--key is required"},
		{name: "missing key file", keyPath: "/nonexistent/key.pem", wantErr: "key"},
		{name: "missing certificate file", keyPath: keyPath, certPath: "/nonexistent/cert.pem", wantErr: "SPIFFE signer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := loadSigner(tt.keyPath, tt.certPath)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.IsType(t, tt.wantType, signer)
		})
	}
}

func TestExpiryNotice(t *testing.T) {
	notAfter := time.Now().Add(certificateLifetime).Truncate(time.Second)
	now := notAfter.Add(-certificateLifetime)
	keyPath, certPath := mintKeyCertFiles(t, testSpiffeID, notAfter)

	spiffeSigner, err := identityv1.NewSpiffeSignerFromFile(keyPath, certPath)
	require.NoError(t, err)

	keySigner, err := identityv1.NewKeySignerFromFile(keyPath)
	require.NoError(t, err)

	shortNotAfter := now.Add(45 * time.Minute)
	shortKeyPath, shortCertPath := mintKeyCertFiles(t, testSpiffeID, shortNotAfter)

	shortSigner, err := identityv1.NewSpiffeSignerFromFile(shortKeyPath, shortCertPath)
	require.NoError(t, err)

	tests := []struct {
		name   string
		signer identityv1.Signer
		want   string
	}{
		{
			name:   "certificate signer reports when the claim stops verifying",
			signer: spiffeSigner,
			want:   "Certificate valid until " + notAfter.UTC().Format(time.RFC3339) + " (" + certificateLifetime.String() + " from now); push the claim again after the certificate is renewed",
		},
		{
			name:   "sub-hour certificate keeps its minutes",
			signer: shortSigner,
			want:   "Certificate valid until " + shortNotAfter.UTC().Format(time.RFC3339) + " (45m0s from now); push the claim again after the certificate is renewed",
		},
		{name: "key signer has nothing to report", signer: keySigner},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, expiryNotice(tt.signer, now))
		})
	}
}
