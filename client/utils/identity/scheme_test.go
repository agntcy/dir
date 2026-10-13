// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemeOf(t *testing.T) {
	tests := []struct {
		subject string
		want    Scheme
		wantOK  bool
	}{
		{subject: "dns:acme.com", want: SchemeDNS, wantOK: true},
		{subject: "acme.com", want: SchemeDNS, wantOK: true},
		{subject: "", want: SchemeDNS, wantOK: true},
		{subject: "did:web:acme.com", want: SchemeDID, wantOK: true},
		{subject: "did:key:z6Mk", want: SchemeDID, wantOK: true},
		{subject: "https://acme.com/agents", want: SchemeWellKnown, wantOK: true},
		{subject: "spiffe://acme.com/agents/finance", want: SchemeSPIFFE, wantOK: true},
		{subject: "ans://v1.0.0.agent.acme.com", want: SchemeANS, wantOK: true},
		{subject: "http://acme.com"},
		{subject: "ftp://acme.com"},
		{subject: "mailto:a@b.c"},
		{subject: "acme.com:8080"},
		{subject: "ANS://v1.0.0.agent.acme.com"},
		{subject: "SPIFFE://acme.com/agent"},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			got, ok := SchemeOf(tt.subject)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSchemeProperties(t *testing.T) {
	tests := []struct {
		scheme          Scheme
		wantPrefix      string
		wantCertificate bool
	}{
		{SchemeDNS, "dns:", false},
		{SchemeDID, "did:", false},
		{SchemeWellKnown, "https://", false},
		{SchemeSPIFFE, "spiffe://", true},
		{SchemeANS, "ans://", true},
		{Scheme("ftp"), "", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.scheme), func(t *testing.T) {
			assert.Equal(t, tt.wantPrefix, tt.scheme.Prefix())
			assert.Equal(t, tt.wantCertificate, tt.scheme.NeedsCertificate())
		})
	}
}

// Every scheme is listed once, and the error text names exactly the schemes
// whose claims carry a certificate.
func TestSchemesAndCertificateSchemes(t *testing.T) {
	require.Equal(t, []Scheme{SchemeDNS, SchemeDID, SchemeWellKnown, SchemeSPIFFE, SchemeANS}, Schemes())
	assert.Equal(t, "spiffe:// and ans://", certificateSchemes())
}

func TestJoinAnd(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  string
	}{
		{name: "none"},
		{name: "one", items: []string{"a"}, want: "a"},
		{name: "two", items: []string{"a", "b"}, want: "a and b"},
		{name: "three", items: []string{"a", "b", "c"}, want: "a, b and c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, joinAnd(tt.items))
		})
	}
}
