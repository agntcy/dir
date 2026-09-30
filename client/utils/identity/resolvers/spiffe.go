// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"context"
	"crypto"
	"errors"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
)

// SPIFFE resolves "spiffe://" subjects. Unlike the other schemes, the key is
// not published anywhere: it comes from the X.509-SVID the claim carries, and
// is only trusted after the certificate validates against the subject's
// trust-domain bundle.
type SPIFFE struct {
	bundles x509bundle.Source
}

// NewSPIFFE creates a SPIFFE resolver trusting the given bundles. A trust
// domain with no bundle fails closed.
func NewSPIFFE(bundles x509bundle.Source) *SPIFFE {
	return &SPIFFE{bundles: bundles}
}

// LoadSPIFFEBundles reads one PEM trust bundle per trust domain, given as a
// map of trust domain (e.g. "acme.com") to bundle file path.
func LoadSPIFFEBundles(trustDomains map[string]string) (*x509bundle.Set, error) {
	loaded := make([]*x509bundle.Bundle, 0, len(trustDomains))

	for domain, path := range trustDomains {
		td, err := spiffeid.TrustDomainFromString(domain)
		if err != nil {
			return nil, fmt.Errorf("trust domain %q: %w", domain, err)
		}

		bundle, err := x509bundle.Load(td, path)
		if err != nil {
			return nil, fmt.Errorf("load trust bundle for %q: %w", domain, err)
		}

		loaded = append(loaded, bundle)
	}

	return x509bundle.NewSet(loaded...), nil
}

// Resolve implements Resolver. certificate is the claim's DER-encoded
// X.509-SVID, which must chain to the subject's trust bundle, carry exactly
// the subject as its SPIFFE ID, and be within its validity window. The claim
// carries only the leaf, so its issuer must be a root in the bundle.
func (s *SPIFFE) Resolve(_ context.Context, subject string, certificate []byte) ([]crypto.PublicKey, error) {
	id, err := spiffeid.FromString(subject)
	if err != nil {
		return nil, fmt.Errorf("invalid spiffe subject %q: %w", subject, err)
	}

	if len(certificate) == 0 {
		return nil, errors.New("spiffe claim carries no certificate")
	}

	certID, chains, err := x509svid.ParseAndVerify([][]byte{certificate}, s.bundles)
	if err != nil {
		return nil, fmt.Errorf("verify SVID for %s: %w", subject, err)
	}

	if certID != id {
		return nil, fmt.Errorf("SVID is for %s, not the claimed subject %s", certID, id)
	}

	pub, ok := toPublicKey(chains[0][0].PublicKey)
	if !ok {
		return nil, fmt.Errorf("SVID for %s has an unsupported key type %T", subject, chains[0][0].PublicKey)
	}

	return []crypto.PublicKey{pub}, nil
}
