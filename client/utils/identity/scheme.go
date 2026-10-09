// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import "strings"

// Scheme is the way a claim's subject names the key material that proves the
// claim. It is read off the subject's spelling, and it decides which resolver
// finds the keys and whether the claim carries a certificate.
type Scheme string

// The schemes a subject can be spelled in.
const (
	// SchemeDNS is "dns:<host>", or a bare host name: the key is published in
	// a DNS TXT record.
	SchemeDNS Scheme = "dns"

	// SchemeDID is "did:web:..." or "did:key:...": the key is in the DID document.
	SchemeDID Scheme = "did"

	// SchemeWellKnown is "https://...": the key is in a well-known JWKS.
	SchemeWellKnown Scheme = "https"

	// SchemeSPIFFE is "spiffe://...": the key is the X.509-SVID the claim carries.
	SchemeSPIFFE Scheme = "spiffe"

	// SchemeANS is "ans://...": the key is the identity certificate the claim
	// carries, attested by the agent's transparency log.
	SchemeANS Scheme = "ans"
)

// schemeSpec is what this package knows about a scheme: the spelling that
// selects it, and whether its claims carry the certificate the key comes from.
type schemeSpec struct {
	scheme      Scheme
	prefix      string
	certificate bool
}

// schemes is the one list every scheme-dependent rule reads. Its order is the
// order messages name the schemes in.
var schemes = []schemeSpec{
	{scheme: SchemeDNS, prefix: "dns:"},
	{scheme: SchemeDID, prefix: "did:"},
	{scheme: SchemeWellKnown, prefix: "https://"},
	{scheme: SchemeSPIFFE, prefix: "spiffe://", certificate: true},
	{scheme: SchemeANS, prefix: "ans://", certificate: true},
}

// Schemes lists every scheme, in the order messages name them.
func Schemes() []Scheme {
	out := make([]Scheme, 0, len(schemes))
	for _, spec := range schemes {
		out = append(out, spec.scheme)
	}

	return out
}

// SchemeOf returns the scheme a subject is spelled in. A subject without a
// scheme is a bare host name and reads as SchemeDNS; one spelled in an unknown
// scheme reports false.
func SchemeOf(subject string) (Scheme, bool) {
	for _, spec := range schemes {
		if strings.HasPrefix(subject, spec.prefix) {
			return spec.scheme, true
		}
	}

	if !strings.Contains(subject, ":") {
		return SchemeDNS, true
	}

	return "", false
}

// Prefix returns the spelling that selects the scheme, e.g. "ans://".
func (s Scheme) Prefix() string {
	return s.spec().prefix
}

// NeedsCertificate reports whether a claim for the scheme carries the
// certificate its key comes from. Every other scheme publishes its key, and a
// claim for it must not carry one.
func (s Scheme) NeedsCertificate() bool {
	return s.spec().certificate
}

func (s Scheme) spec() schemeSpec {
	for _, spec := range schemes {
		if spec.scheme == s {
			return spec
		}
	}

	return schemeSpec{}
}

// certificateSchemes names the schemes whose claims carry a certificate, for
// messages: "spiffe:// and ans://".
func certificateSchemes() string {
	var prefixes []string

	for _, spec := range schemes {
		if spec.certificate {
			prefixes = append(prefixes, spec.prefix)
		}
	}

	return joinAnd(prefixes)
}

// joinAnd lists items the way prose does: "a", "a and b", "a, b and c".
func joinAnd(items []string) string {
	if len(items) <= 1 {
		return strings.Join(items, "")
	}

	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}
