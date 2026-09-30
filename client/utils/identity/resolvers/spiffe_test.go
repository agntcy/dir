// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"crypto"
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

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/require"
)

const svidSubject = "spiffe://acme.com/agents/finance"

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newTestCA(t *testing.T, name string) *testCA {
	t.Helper()

	key := newECKey(t, elliptic.P256())
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &testCA{cert: cert, key: key}
}

type svidOpts struct {
	uris      []string
	notBefore time.Time
	notAfter  time.Time
	isCA      bool
}

// issue returns the DER of an SVID for key signed by the CA.
func (ca *testCA) issue(t *testing.T, key crypto.Signer, opts svidOpts) []byte {
	t.Helper()

	if opts.notBefore.IsZero() {
		opts.notBefore = time.Now().Add(-time.Minute)
	}

	if opts.notAfter.IsZero() {
		opts.notAfter = time.Now().Add(time.Hour)
	}

	uris := make([]*url.URL, 0, len(opts.uris))

	for _, u := range opts.uris {
		parsed, err := url.Parse(u)
		require.NoError(t, err)

		uris = append(uris, parsed)
	}

	usage := x509.KeyUsageDigitalSignature
	if opts.isCA {
		usage = x509.KeyUsageCertSign
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "svid"},
		NotBefore:             opts.notBefore,
		NotAfter:              opts.notAfter,
		URIs:                  uris,
		KeyUsage:              usage,
		IsCA:                  opts.isCA,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, key.Public(), ca.key)
	require.NoError(t, err)

	return der
}

func bundleFor(t *testing.T, trustDomain string, cas ...*testCA) *x509bundle.Set {
	t.Helper()

	td, err := spiffeid.TrustDomainFromString(trustDomain)
	require.NoError(t, err)

	certs := make([]*x509.Certificate, 0, len(cas))
	for _, ca := range cas {
		certs = append(certs, ca.cert)
	}

	return x509bundle.NewSet(x509bundle.FromX509Authorities(td, certs))
}

func TestSPIFFE_Resolve(t *testing.T) {
	ca := newTestCA(t, "acme root")
	key := newECKey(t, elliptic.P256())

	resolver := NewSPIFFE(bundleFor(t, "acme.com", ca))

	keys, err := resolver.Resolve(t.Context(), svidSubject, ca.issue(t, key, svidOpts{uris: []string{svidSubject}}))
	require.NoError(t, err)
	requireSameKeys(t, []crypto.PublicKey{&key.PublicKey}, keys)
}

func TestSPIFFE_Resolve_FailsClosed(t *testing.T) {
	ca := newTestCA(t, "acme root")
	otherCA := newTestCA(t, "other root")
	key := newECKey(t, elliptic.P256())
	resolver := NewSPIFFE(bundleFor(t, "acme.com", ca))

	valid := ca.issue(t, key, svidOpts{uris: []string{svidSubject}})

	tests := []struct {
		name        string
		resolver    *SPIFFE
		subject     string
		certificate []byte
		wantErr     string
	}{
		{
			name:        "no bundle for the trust domain",
			resolver:    NewSPIFFE(bundleFor(t, "unrelated.org", ca)),
			subject:     svidSubject,
			certificate: valid,
			wantErr:     "verify SVID",
		},
		{
			name:        "empty bundle set",
			resolver:    NewSPIFFE(x509bundle.NewSet()),
			subject:     svidSubject,
			certificate: valid,
			wantErr:     "verify SVID",
		},
		{
			name:        "issued by an untrusted CA",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: otherCA.issue(t, key, svidOpts{uris: []string{svidSubject}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate is for a different SPIFFE ID",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.issue(t, key, svidOpts{uris: []string{"spiffe://acme.com/agents/other"}}),
			wantErr:     "not the claimed subject",
		},
		{
			name:        "certificate is for another trust domain",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.issue(t, key, svidOpts{uris: []string{"spiffe://evil.org/agents/finance"}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "expired certificate",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.issue(t, key, svidOpts{uris: []string{svidSubject}, notBefore: time.Now().Add(-2 * time.Hour), notAfter: time.Now().Add(-time.Hour)}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate not yet valid",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.issue(t, key, svidOpts{uris: []string{svidSubject}, notBefore: time.Now().Add(time.Hour), notAfter: time.Now().Add(2 * time.Hour)}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate has no SPIFFE ID",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.issue(t, key, svidOpts{}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate has several SPIFFE IDs",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.issue(t, key, svidOpts{uris: []string{svidSubject, "spiffe://acme.com/agents/other"}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "CA certificate used as a leaf",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.issue(t, key, svidOpts{uris: []string{svidSubject}, isCA: true}),
			wantErr:     "verify SVID",
		},
		{name: "no certificate", resolver: resolver, subject: svidSubject, wantErr: "no certificate"},
		{name: "garbage certificate", resolver: resolver, subject: svidSubject, certificate: []byte("garbage"), wantErr: "verify SVID"},
		{name: "not a SPIFFE subject", resolver: resolver, subject: "dns:acme.com", certificate: valid, wantErr: "invalid spiffe subject"},
		{name: "empty subject", resolver: resolver, subject: "", certificate: valid, wantErr: "invalid spiffe subject"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, err := tt.resolver.Resolve(t.Context(), tt.subject, tt.certificate)
			require.ErrorContains(t, err, tt.wantErr)
			require.Empty(t, keys)
		})
	}
}

func TestLoadSPIFFEBundles(t *testing.T) {
	ca := newTestCA(t, "acme root")
	dir := t.TempDir()

	bundlePath := filepath.Join(dir, "acme.pem")
	require.NoError(t, os.WriteFile(bundlePath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw}), 0o600))

	set, err := LoadSPIFFEBundles(map[string]string{"acme.com": bundlePath})
	require.NoError(t, err)

	key := newECKey(t, elliptic.P256())

	keys, err := NewSPIFFE(set).Resolve(t.Context(), svidSubject, ca.issue(t, key, svidOpts{uris: []string{svidSubject}}))
	require.NoError(t, err)
	require.Len(t, keys, 1)

	// Errors are surfaced, never silently skipped.
	_, err = LoadSPIFFEBundles(map[string]string{"acme.com": filepath.Join(dir, "missing.pem")})
	require.ErrorContains(t, err, "load trust bundle")

	_, err = LoadSPIFFEBundles(map[string]string{"not a domain!": bundlePath})
	require.ErrorContains(t, err, "trust domain")

	empty, err := LoadSPIFFEBundles(nil)
	require.NoError(t, err)

	_, err = NewSPIFFE(empty).Resolve(t.Context(), svidSubject, ca.issue(t, key, svidOpts{uris: []string{svidSubject}}))
	require.Error(t, err, "no configured bundles must fail closed")
}
