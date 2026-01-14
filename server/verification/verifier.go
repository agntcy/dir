// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"context"
	"errors"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

var verifierLogger = logging.Logger("verification/verifier")

// Verifier handles name ownership verification for OASF records.
// It supports DNS TXT records and well-known files based on the protocol prefix.
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

// NewVerifier creates a new verifier with the given options.
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

// Verify checks if the given signing key is authorized for the name.
// It parses the protocol prefix from the record name to determine the verification method:
//   - dns://domain/path -> use DNS TXT records only
//   - wellknown://domain/path -> use well-known file only
//   - domain/path -> try DNS first, then fall back to well-known
func (v *Verifier) Verify(ctx context.Context, recordName string, signingKey []byte) *Result {
	result := &Result{
		VerifiedAt: time.Now(),
	}

	// Parse the record name
	parsed := ParseName(recordName)
	if parsed == nil {
		result.Error = "could not parse record name"

		verifierLogger.Debug("Name parsing failed", "recordName", recordName)

		return result
	}

	result.Domain = parsed.Domain
	verifierLogger.Debug("Verifying name ownership",
		"domain", parsed.Domain,
		"protocol", parsed.Protocol,
		"recordName", recordName)

	// Look up keys for the domain based on protocol
	keys, method, err := v.lookupKeys(ctx, parsed)
	if err != nil {
		result.Error = err.Error()
		verifierLogger.Debug("Key lookup failed", "domain", parsed.Domain, "error", err)

		return result
	}

	if len(keys) == 0 {
		result.Error = "no OASF keys found for domain"
		result.Method = string(MethodNone)

		verifierLogger.Debug("No keys found for domain", "domain", parsed.Domain)

		return result
	}

	result.Method = string(method)

	// Check if signing key matches any domain key
	matchedKey, matched := MatchKey(signingKey, keys)
	if !matched {
		result.Error = "signing key does not match any domain key"

		verifierLogger.Debug("Key mismatch", "domain", parsed.Domain, "domainKeyCount", len(keys))

		return result
	}

	// Success!
	result.Verified = true
	result.MatchedKeyID = matchedKey.ID

	verifierLogger.Info("Name ownership verified",
		"domain", parsed.Domain,
		"method", method,
		"keyID", matchedKey.ID,
		"keyType", matchedKey.Type)

	return result
}

// lookupKeys retrieves the public keys for a domain based on the protocol.
// If no protocol prefix is specified, no verification is performed.
func (v *Verifier) lookupKeys(ctx context.Context, parsed *ParsedName) ([]PublicKey, VerificationMethod, error) {
	switch parsed.Protocol {
	case DNSProtocol:
		// DNS only
		keys, err := v.dns.LookupKeys(ctx, parsed.Domain)
		if err != nil {
			return nil, MethodNone, err
		}

		return keys, MethodDNS, nil

	case WellKnownProtocol:
		// Well-known only
		keys, err := v.wellKnown.FetchKeys(ctx, parsed.Domain)
		if err != nil {
			return nil, MethodNone, err
		}

		return keys, MethodWellKnown, nil

	default:
		// No protocol prefix - skip verification
		return nil, MethodNone, errors.New("no verification protocol specified in name (use dns:// or wellknown:// prefix)")
	}
}
