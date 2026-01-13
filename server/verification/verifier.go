// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"context"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

var verifierLogger = logging.Logger("verification/verifier")

// Verifier handles domain ownership verification for OASF records.
// It checks DNS TXT records first, then falls back to well-known files.
type Verifier struct {
	// dns is the DNS resolver for TXT record lookups.
	dns *DNSResolver

	// wellKnown is the fetcher for well-known files.
	wellKnown *WellKnownFetcher

	// allowInsecure allows HTTP instead of HTTPS for well-known fetching (testing only).
	allowInsecure bool
}

// VerifierOption configures a Verifier.
type VerifierOption func(*Verifier)

// WithDNSResolver sets a custom DNS resolver.
func WithDNSResolver(dns *DNSResolver) VerifierOption {
	return func(v *Verifier) {
		v.dns = dns
	}
}

// WithWellKnownFetcher sets a custom well-known fetcher.
func WithWellKnownFetcher(wk *WellKnownFetcher) VerifierOption {
	return func(v *Verifier) {
		v.wellKnown = wk
	}
}

// WithAllowInsecureWellKnown allows HTTP instead of HTTPS for well-known fetching.
// WARNING: Only use for local development/testing. Never enable in production.
func WithAllowInsecureWellKnown(allow bool) VerifierOption {
	return func(v *Verifier) {
		v.allowInsecure = allow
	}
}

// NewVerifier creates a new domain verifier with the given options.
func NewVerifier(opts ...VerifierOption) *Verifier {
	v := &Verifier{}

	for _, opt := range opts {
		opt(v)
	}

	// Create default DNS resolver if not provided
	if v.dns == nil {
		v.dns = NewDNSResolver()
	}

	// Create default well-known fetcher if not provided
	if v.wellKnown == nil {
		v.wellKnown = NewWellKnownFetcher(WithAllowInsecure(v.allowInsecure))
	}

	return v
}

// Verify checks if the given signing key is authorized for the domain.
// It extracts the domain from the record name, looks up the domain's published keys,
// and checks if the signing key matches any of them.
func (v *Verifier) Verify(ctx context.Context, recordName string, signingKey []byte) *Result {
	result := &Result{
		VerifiedAt: time.Now(),
	}

	// Extract domain from record name
	domain := ExtractDomain(recordName)
	if domain == "" {
		result.Error = "could not extract domain from record name"
		verifierLogger.Debug("Domain extraction failed", "recordName", recordName)

		return result
	}

	result.Domain = domain
	verifierLogger.Debug("Verifying domain ownership", "domain", domain, "recordName", recordName)

	// Look up keys for the domain
	keys, method, err := v.lookupKeys(ctx, domain)
	if err != nil {
		result.Error = err.Error()
		verifierLogger.Debug("Key lookup failed", "domain", domain, "error", err)

		return result
	}

	if len(keys) == 0 {
		result.Error = "no OASF keys found for domain"
		result.Method = string(MethodNone)
		verifierLogger.Debug("No keys found for domain", "domain", domain)

		return result
	}

	result.Method = string(method)

	// Check if signing key matches any domain key
	matchedKey, matched := MatchKey(signingKey, keys)
	if !matched {
		result.Error = "signing key does not match any domain key"
		verifierLogger.Debug("Key mismatch", "domain", domain, "domainKeyCount", len(keys))

		return result
	}

	// Success!
	result.Verified = true
	result.MatchedKeyID = matchedKey.ID

	verifierLogger.Info("Domain ownership verified",
		"domain", domain,
		"method", method,
		"keyID", matchedKey.ID,
		"keyType", matchedKey.Type)

	return result
}

// lookupKeys retrieves the public keys for a domain.
// It tries DNS TXT records first, then falls back to well-known files.
func (v *Verifier) lookupKeys(ctx context.Context, domain string) ([]PublicKey, VerificationMethod, error) {
	// Try DNS first
	keys, err := v.dns.LookupKeys(ctx, domain)
	if err == nil && len(keys) > 0 {
		return keys, MethodDNS, nil
	}

	if err != nil {
		verifierLogger.Debug("DNS lookup failed, trying well-known", "domain", domain, "error", err)
	} else {
		verifierLogger.Debug("No DNS records, trying well-known", "domain", domain)
	}

	// Fall back to well-known file
	keys, err = v.wellKnown.FetchKeys(ctx, domain)
	if err != nil {
		return nil, MethodNone, err
	}

	if len(keys) > 0 {
		return keys, MethodWellKnown, nil
	}

	// No keys found via either method
	return nil, MethodNone, nil
}
