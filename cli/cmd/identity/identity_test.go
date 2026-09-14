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

func TestCommand_HasExpectedSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command.Commands() {
		names[c.Name()] = true
	}

	assert.True(t, names["claim"])
	assert.True(t, names["status"])
	assert.True(t, names["resolve"])
}

// mintKeyCertFiles writes a P-256 key and a self-signed certificate carrying
// uri as its URI SAN into a temporary directory and returns their paths.
func mintKeyCertFiles(t *testing.T, uri string) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}

	if uri != "" {
		parsed, err := url.Parse(uri)
		require.NoError(t, err)

		tmpl.URIs = []*url.URL{parsed}
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
	tests := []struct {
		name     string
		uri      string
		withCert bool
		keyPath  string
		wantType any
		wantErr  string
	}{
		{name: "key alone builds a key signer", wantType: &identityv1.KeySigner{}},
		{name: "spiffe certificate selects the spiffe signer", uri: "spiffe://acme.com/agents/finance", withCert: true, wantType: &identityv1.SpiffeSigner{}},
		{name: "ans certificate selects the ans signer", uri: "ans://v1.0.0.agent.example.com", withCert: true, wantType: &identityv1.AnsSigner{}},
		{name: "certificate with neither scheme", uri: "https://acme.com", withCert: true, wantErr: "spiffe:// or ans://"},
		{name: "key is required", keyPath: "-", wantErr: "--key is required"},
		{name: "missing key file", keyPath: "/nonexistent/key.pem", wantErr: "key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath, certPath := mintKeyCertFiles(t, tt.uri)

			switch tt.keyPath {
			case "-":
				keyPath = ""
			case "":
			default:
				keyPath = tt.keyPath
			}

			if !tt.withCert {
				certPath = ""
			}

			signer, err := loadSigner(keyPath, certPath)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.IsType(t, tt.wantType, signer)
		})
	}
}

// The claim command's flags: --key is required on its own, --cert is optional
// and needs --key. #2125 declared the pair required together, which rejected
// every plain-key claim.
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

// resetFlags clears the package-level flag set between cases; pflag keeps the
// Changed state of a flag across ParseFlags calls.
func resetFlags(t *testing.T, flags *pflag.FlagSet) {
	t.Helper()

	flags.VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		require.NoError(t, f.Value.Set(f.DefValue))
	})
}

func TestExpiryNotice(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	keyPath, certPath := mintKeyCertFiles(t, "ans://v1.0.0.agent.example.com")

	certSigner, err := identityv1.NewCertificateSignerFromFile(keyPath, certPath)
	require.NoError(t, err)

	keySigner, err := identityv1.NewKeySignerFromFile(keyPath)
	require.NoError(t, err)

	tests := []struct {
		name   string
		signer identityv1.Signer
		want   string
	}{
		{name: "certificate signers report when the claim stops verifying", signer: certSigner, want: "valid until"},
		{name: "key signers have nothing to report", signer: keySigner, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expiryNotice(tt.signer, now)
			if tt.want == "" {
				assert.Empty(t, got)

				return
			}

			assert.Contains(t, got, tt.want)
			assert.Contains(t, got, "renewed")
		})
	}
}
