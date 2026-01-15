// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

var providerLogger = logging.Logger("naming/provider")

// KeyLookup defines the interface for looking up public keys for a domain.
type KeyLookup interface {
	// LookupKeys retrieves public keys for the given domain.
	LookupKeys(ctx context.Context, domain string) ([]PublicKey, error)
}

// KeyLookupWithScheme defines the interface for looking up public keys with a URL scheme.
type KeyLookupWithScheme interface {
	// LookupKeysWithScheme retrieves public keys for the given domain using the specified scheme.
	LookupKeysWithScheme(ctx context.Context, domain, scheme string) ([]PublicKey, error)
}

// Provider handles name ownership verification for OASF records.
// It routes to the appropriate verification method based on the protocol prefix.
type Provider struct {
	// dns is the DNS resolver for TXT record lookups.
	dns KeyLookup

	// wellKnown is the fetcher for JWKS well-known files.
	wellKnown KeyLookupWithScheme
}

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// WithDNSLookup sets the DNS key lookup implementation.
func WithDNSLookup(dns KeyLookup) ProviderOption {
	return func(p *Provider) {
		p.dns = dns
	}
}

// WithWellKnownLookup sets the well-known key lookup implementation.
func WithWellKnownLookup(wk KeyLookupWithScheme) ProviderOption {
	return func(p *Provider) {
		p.wellKnown = wk
	}
}

// NewProvider creates a new naming provider with the given options.
func NewProvider(opts ...ProviderOption) *Provider {
	p := &Provider{}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Verify checks if the given signing key is authorized for the name.
// It parses the protocol prefix from the record name to determine the verification method:
//   - dns://domain/path -> use DNS TXT records
//   - https://domain/path -> use JWKS well-known file via HTTPS
//   - http://domain/path -> use JWKS well-known file via HTTP (testing only)
//   - domain/path -> no verification (protocol prefix required)
func (p *Provider) Verify(ctx context.Context, recordName string, signingKey []byte) *Result {
	result := &Result{
		VerifiedAt: time.Now(),
	}

	// Parse the record name
	parsed := ParseName(recordName)
	if parsed == nil {
		result.Error = "could not parse record name"

		providerLogger.Debug("Name parsing failed", "recordName", recordName)

		return result
	}

	result.Domain = parsed.Domain
	providerLogger.Debug("Verifying name ownership",
		"domain", parsed.Domain,
		"protocol", parsed.Protocol,
		"recordName", recordName)

	// Look up keys for the domain based on protocol
	keys, method, err := p.lookupKeys(ctx, parsed)
	if err != nil {
		result.Error = err.Error()
		providerLogger.Debug("Key lookup failed", "domain", parsed.Domain, "error", err)

		return result
	}

	if len(keys) == 0 {
		result.Error = "no keys found for domain"
		result.Method = string(MethodNone)

		providerLogger.Debug("No keys found for domain", "domain", parsed.Domain)

		return result
	}

	result.Method = string(method)

	// Check if signing key matches any domain key
	matchedKey, matched := MatchKey(signingKey, keys)
	if !matched {
		result.Error = "signing key does not match any domain key"

		providerLogger.Debug("Key mismatch", "domain", parsed.Domain, "domainKeyCount", len(keys))

		return result
	}

	// Success!
	result.Verified = true
	result.MatchedKeyID = matchedKey.ID

	providerLogger.Info("Name ownership verified",
		"domain", parsed.Domain,
		"method", method,
		"keyID", matchedKey.ID,
		"keyType", matchedKey.Type)

	return result
}

// lookupKeys retrieves the public keys for a domain based on the protocol.
// If no protocol prefix is specified, no verification is performed.
func (p *Provider) lookupKeys(ctx context.Context, parsed *ParsedName) ([]PublicKey, VerificationMethod, error) {
	switch parsed.Protocol {
	case DNSProtocol:
		if p.dns == nil {
			return nil, MethodNone, errors.New("DNS verification not configured")
		}

		keys, err := p.dns.LookupKeys(ctx, parsed.Domain)
		if err != nil {
			return nil, MethodNone, fmt.Errorf("DNS key lookup failed: %w", err)
		}

		return keys, MethodDNS, nil

	case HTTPSProtocol:
		if p.wellKnown == nil {
			return nil, MethodNone, errors.New("JWKS verification not configured")
		}

		keys, err := p.wellKnown.LookupKeysWithScheme(ctx, parsed.Domain, "https")
		if err != nil {
			return nil, MethodNone, fmt.Errorf("JWKS lookup failed: %w", err)
		}

		return keys, MethodWellKnown, nil

	case HTTPProtocol:
		if p.wellKnown == nil {
			return nil, MethodNone, errors.New("JWKS verification not configured")
		}

		keys, err := p.wellKnown.LookupKeysWithScheme(ctx, parsed.Domain, "http")
		if err != nil {
			return nil, MethodNone, fmt.Errorf("JWKS lookup failed: %w", err)
		}

		return keys, MethodWellKnown, nil

	default:
		// No protocol prefix - skip verification
		return nil, MethodNone, errors.New("no verification protocol specified in name (use dns://, https://, or http:// prefix)")
	}
}
