// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package spifferesolver

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

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/internal/testutil"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/require"
)

const svidSubject = testutil.SVIDSubject

func TestSPIFFE_Resolve(t *testing.T) {
	ca := newCA(t, "acme root")
	key := testutil.NewECKey(t, elliptic.P256())

	resolver := New(bundleFor(t, "acme.com", ca))

	keys, err := resolver.Resolve(t.Context(), svidSubject, ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}}))
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&key.PublicKey}, keys)
}

func TestSPIFFE_Resolve_FailsClosed(t *testing.T) {
	ca := newCA(t, "acme root")
	otherCA := newCA(t, "other root")
	key := testutil.NewECKey(t, elliptic.P256())
	resolver := New(bundleFor(t, "acme.com", ca))

	valid := ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}})

	tests := []struct {
		name        string
		resolver    *Resolver
		subject     string
		certificate []byte
		wantErr     string
	}{
		{
			name:        "no bundle for the trust domain",
			resolver:    New(bundleFor(t, "unrelated.org", ca)),
			subject:     svidSubject,
			certificate: valid,
			wantErr:     "verify SVID",
		},
		{
			name:        "empty bundle set",
			resolver:    New(x509bundle.NewSet()),
			subject:     svidSubject,
			certificate: valid,
			wantErr:     "verify SVID",
		},
		{
			name:        "issued by an untrusted CA",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: otherCA.Issue(t, key, svidOptions{URIs: []string{svidSubject}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate is for a different SPIFFE ID",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, svidOptions{URIs: []string{"spiffe://acme.com/agents/other"}}),
			wantErr:     "not the claimed subject",
		},
		{
			name:        "certificate is for another trust domain",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, svidOptions{URIs: []string{"spiffe://evil.org/agents/finance"}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "expired certificate",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}, NotBefore: time.Now().Add(-2 * time.Hour), NotAfter: time.Now().Add(-time.Hour)}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate not yet valid",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}, NotBefore: time.Now().Add(time.Hour), NotAfter: time.Now().Add(2 * time.Hour)}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate has no SPIFFE ID",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, svidOptions{}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate has several SPIFFE IDs",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, svidOptions{URIs: []string{svidSubject, "spiffe://acme.com/agents/other"}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "CA certificate used as a leaf",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}, IsCA: true}),
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

func TestLoadBundles(t *testing.T) {
	ca := newCA(t, "acme root")
	dir := t.TempDir()

	bundlePath := filepath.Join(dir, "acme.pem")
	require.NoError(t, os.WriteFile(bundlePath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw}), 0o600))

	set, err := LoadBundles(map[string]string{"acme.com": bundlePath})
	require.NoError(t, err)

	key := testutil.NewECKey(t, elliptic.P256())

	keys, err := New(set).Resolve(t.Context(), svidSubject, ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}}))
	require.NoError(t, err)
	require.Len(t, keys, 1)

	// Errors are surfaced, never silently skipped.
	_, err = LoadBundles(map[string]string{"acme.com": filepath.Join(dir, "missing.pem")})
	require.ErrorContains(t, err, "load trust bundle")

	_, err = LoadBundles(map[string]string{"not a domain!": bundlePath})
	require.ErrorContains(t, err, "trust domain")

	// A bundle that cannot be read leaves only its own domain out of the set.
	partial, err := LoadBundles(map[string]string{"acme.com": bundlePath, "other.org": filepath.Join(dir, "missing.pem")})
	require.ErrorContains(t, err, "other.org")

	keys, err = New(partial).Resolve(t.Context(), svidSubject, ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}}))
	require.NoError(t, err)
	require.Len(t, keys, 1)

	empty, err := LoadBundles(nil)
	require.NoError(t, err)

	_, err = New(empty).Resolve(t.Context(), svidSubject, ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}}))
	require.Error(t, err, "no configured bundles must fail closed")
}

// testCA is a throwaway certificate authority for issuing test SVIDs.
type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newCA(t *testing.T, name string) *testCA {
	t.Helper()

	key := testutil.NewECKey(t, elliptic.P256())
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

// svidOptions shapes a certificate issued by testCA.Issue.
type svidOptions struct {
	URIs      []string
	NotBefore time.Time
	NotAfter  time.Time
	IsCA      bool
}

// Issue returns the DER of a certificate for key signed by the CA. By
// default it is valid from a minute ago for an hour.
func (ca *testCA) Issue(t *testing.T, key crypto.Signer, opts svidOptions) []byte {
	t.Helper()

	if opts.NotBefore.IsZero() {
		opts.NotBefore = time.Now().Add(-time.Minute)
	}

	if opts.NotAfter.IsZero() {
		opts.NotAfter = time.Now().Add(time.Hour)
	}

	uris := make([]*url.URL, 0, len(opts.URIs))

	for _, u := range opts.URIs {
		parsed, err := url.Parse(u)
		require.NoError(t, err)

		uris = append(uris, parsed)
	}

	usage := x509.KeyUsageDigitalSignature
	if opts.IsCA {
		usage = x509.KeyUsageCertSign
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "svid"},
		NotBefore:             opts.NotBefore,
		NotAfter:              opts.NotAfter,
		URIs:                  uris,
		KeyUsage:              usage,
		IsCA:                  opts.IsCA,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, key.Public(), ca.key)
	require.NoError(t, err)

	return der
}

// bundleFor returns a trust bundle set for trustDomain rooted at cas.
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

func TestResolver_SatisfiesContract(t *testing.T) {
	var _ resolvers.Resolver = New(x509bundle.NewSet())
}

func TestEndToEnd(t *testing.T) {
	ca := newCA(t, "acme root")
	key := testutil.NewECKey(t, elliptic.P256())

	svid := testutil.CertificatePEM(ca.Issue(t, key, svidOptions{URIs: []string{svidSubject}}))

	claim := testutil.SignedCertClaim(t, svidSubject, key, svid)

	ok, err := testutil.VerifyClaim(t, New(bundleFor(t, "acme.com", ca)), claim)
	require.NoError(t, err)
	require.True(t, ok)

	// With no trust bundle for the subject's domain, the same claim fails closed.
	ok, err = testutil.VerifyClaim(t, New(bundleFor(t, "unrelated.org", ca)), claim)
	require.Error(t, err)
	require.False(t, ok)
}
