// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package spiffe loads per-trust-domain CA bundles used to validate the
// X.509-SVID certificates embedded in SPIFFE identity/ownership claims.
// SPIFFE claims are verified directly in api/identity/v1 against the
// certificates this package returns - it is not a server/identity.Resolver.
package spiffe

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

// Config maps SPIFFE trust domains to a PEM file containing their trust
// bundle CA certificate(s).
type Config struct {
	// TrustDomains maps a trust domain (e.g. "acme.com") to a path containing
	// one or more PEM-encoded CA certificates.
	TrustDomains map[string]string `json:"trust_domains,omitempty" mapstructure:"trust_domains"`
}

// Bundles holds loaded trust bundle certificates, keyed by trust domain.
type Bundles struct {
	certs map[string][]*x509.Certificate
}

// NewBundles builds a Bundles directly from already-parsed certificates,
// keyed by trust domain. Useful for tests; production code should use Load.
func NewBundles(certs map[string][]*x509.Certificate) *Bundles {
	return &Bundles{certs: certs}
}

// Load reads and parses all configured trust bundle files.
func Load(cfg Config) (*Bundles, error) {
	b := &Bundles{certs: make(map[string][]*x509.Certificate, len(cfg.TrustDomains))}

	for domain, path := range cfg.TrustDomains {
		certs, err := loadCertsFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("load trust bundle for %q: %w", domain, err)
		}

		b.certs[domain] = certs
	}

	return b, nil
}

// TrustedCerts returns the trusted CA certificates for spiffeID's trust
// domain, or nil if the trust domain has no configured bundle. Callers (see
// api/identity/v1.VerifyOwnershipClaim/VerifyIdentityClaim) treat a nil/empty
// result as verification failure - a SPIFFE claim always requires an
// anchored trust bundle.
func (b *Bundles) TrustedCerts(spiffeID string) []*x509.Certificate {
	return b.certs[trustDomainFromSpiffeID(spiffeID)]
}

// trustDomainFromSpiffeID extracts the trust domain from a "spiffe://<trust-domain>/..." ID.
func trustDomainFromSpiffeID(spiffeID string) string {
	rest := strings.TrimPrefix(spiffeID, "spiffe://")
	if idx := strings.IndexByte(rest, '/'); idx >= 0 {
		return rest[:idx]
	}

	return rest
}

// loadCertsFromFile reads all PEM-encoded CERTIFICATE blocks from path.
func loadCertsFromFile(path string) ([]*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var certs []*x509.Certificate

	for len(data) > 0 {
		var block *pem.Block

		block, data = pem.Decode(data)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate: %w", err)
		}

		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in %q", path)
	}

	return certs, nil
}
