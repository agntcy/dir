// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package spifferesolver

import (
	"crypto"
	"crypto/elliptic"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agntcy/dir/client/utils/identity/resolvers/internal/testutil"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/stretchr/testify/require"
)

const svidSubject = "spiffe://acme.com/agents/finance"

func TestSPIFFE_Resolve(t *testing.T) {
	ca := testutil.NewCA(t, "acme root")
	key := testutil.NewECKey(t, elliptic.P256())

	resolver := New(testutil.Bundle(t, "acme.com", ca))

	keys, err := resolver.Resolve(t.Context(), svidSubject, ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}}))
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&key.PublicKey}, keys)
}

func TestSPIFFE_Resolve_FailsClosed(t *testing.T) {
	ca := testutil.NewCA(t, "acme root")
	otherCA := testutil.NewCA(t, "other root")
	key := testutil.NewECKey(t, elliptic.P256())
	resolver := New(testutil.Bundle(t, "acme.com", ca))

	valid := ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}})

	tests := []struct {
		name        string
		resolver    *Resolver
		subject     string
		certificate []byte
		wantErr     string
	}{
		{
			name:        "no bundle for the trust domain",
			resolver:    New(testutil.Bundle(t, "unrelated.org", ca)),
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
			certificate: otherCA.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate is for a different SPIFFE ID",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{"spiffe://acme.com/agents/other"}}),
			wantErr:     "not the claimed subject",
		},
		{
			name:        "certificate is for another trust domain",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{"spiffe://evil.org/agents/finance"}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "expired certificate",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}, NotBefore: time.Now().Add(-2 * time.Hour), NotAfter: time.Now().Add(-time.Hour)}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate not yet valid",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}, NotBefore: time.Now().Add(time.Hour), NotAfter: time.Now().Add(2 * time.Hour)}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate has no SPIFFE ID",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, testutil.SVIDOptions{}),
			wantErr:     "verify SVID",
		},
		{
			name:        "certificate has several SPIFFE IDs",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject, "spiffe://acme.com/agents/other"}}),
			wantErr:     "verify SVID",
		},
		{
			name:        "CA certificate used as a leaf",
			resolver:    resolver,
			subject:     svidSubject,
			certificate: ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}, IsCA: true}),
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
	ca := testutil.NewCA(t, "acme root")
	dir := t.TempDir()

	bundlePath := filepath.Join(dir, "acme.pem")
	require.NoError(t, os.WriteFile(bundlePath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Cert.Raw}), 0o600))

	set, err := LoadBundles(map[string]string{"acme.com": bundlePath})
	require.NoError(t, err)

	key := testutil.NewECKey(t, elliptic.P256())

	keys, err := New(set).Resolve(t.Context(), svidSubject, ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}}))
	require.NoError(t, err)
	require.Len(t, keys, 1)

	// Errors are surfaced, never silently skipped.
	_, err = LoadBundles(map[string]string{"acme.com": filepath.Join(dir, "missing.pem")})
	require.ErrorContains(t, err, "load trust bundle")

	_, err = LoadBundles(map[string]string{"not a domain!": bundlePath})
	require.ErrorContains(t, err, "trust domain")

	empty, err := LoadBundles(nil)
	require.NoError(t, err)

	_, err = New(empty).Resolve(t.Context(), svidSubject, ca.Issue(t, key, testutil.SVIDOptions{URIs: []string{svidSubject}}))
	require.Error(t, err, "no configured bundles must fail closed")
}
