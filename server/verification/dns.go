// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

const (
	// defaultDNSTimeout is the default timeout for DNS lookups.
	defaultDNSTimeout = 5 * time.Second
	// maxTruncateLen is the maximum length for truncated log strings.
	maxTruncateLen = 50
)

var dnsLogger = logging.Logger("verification/dns")

// DNSResolver handles DNS TXT record lookups for domain verification.
type DNSResolver struct {
	// resolver is the underlying DNS resolver (nil uses default).
	resolver *net.Resolver

	// timeout is the maximum time to wait for DNS resolution.
	timeout time.Duration
}

// DNSResolverOption configures a DNSResolver.
type DNSResolverOption func(*DNSResolver)

// WithDNSTimeout sets the DNS resolution timeout.
func WithDNSTimeout(timeout time.Duration) DNSResolverOption {
	return func(r *DNSResolver) {
		r.timeout = timeout
	}
}

// WithResolver sets a custom DNS resolver.
func WithResolver(resolver *net.Resolver) DNSResolverOption {
	return func(r *DNSResolver) {
		r.resolver = resolver
	}
}

// NewDNSResolver creates a new DNS resolver with the given options.
func NewDNSResolver(opts ...DNSResolverOption) *DNSResolver {
	r := &DNSResolver{
		timeout: defaultDNSTimeout,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// LookupKeys retrieves public keys from DNS TXT records for the given domain.
// It looks up _oasf.<domain> and parses any OASF-formatted TXT records.
func (r *DNSResolver) LookupKeys(ctx context.Context, domain string) ([]PublicKey, error) {
	// Create context with timeout
	lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Build the DNS name to lookup
	dnsName := DNSRecordPrefix + domain

	dnsLogger.Debug("Looking up DNS TXT records", "domain", domain, "dnsName", dnsName)

	// Perform DNS TXT lookup
	var records []string

	var err error

	if r.resolver != nil {
		records, err = r.resolver.LookupTXT(lookupCtx, dnsName)
	} else {
		records, err = net.DefaultResolver.LookupTXT(lookupCtx, dnsName)
	}

	if err != nil {
		// Check if it's a "not found" error (NXDOMAIN)
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			if dnsErr.IsNotFound {
				dnsLogger.Debug("No DNS TXT records found", "domain", domain)

				return nil, nil // Not an error, just no records
			}
		}

		return nil, fmt.Errorf("DNS lookup failed for %s: %w", dnsName, err)
	}

	dnsLogger.Debug("Found DNS TXT records", "domain", domain, "count", len(records))

	// Parse OASF records
	keys := make([]PublicKey, 0, len(records))

	for _, record := range records {
		// Skip non-OASF records
		if !isOASFRecord(record) {
			dnsLogger.Debug("Skipping non-OASF TXT record", "record", truncateString(record, maxTruncateLen))

			continue
		}

		key, err := ParseDNSTXTRecord(record)
		if err != nil {
			dnsLogger.Warn("Failed to parse OASF TXT record", "record", truncateString(record, maxTruncateLen), "error", err)

			continue
		}

		keys = append(keys, *key)
		dnsLogger.Debug("Parsed OASF public key from DNS", "domain", domain, "keyType", key.Type)
	}

	return keys, nil
}

// isOASFRecord checks if a TXT record appears to be an OASF record.
func isOASFRecord(record string) bool {
	// OASF records start with "v=oasf"
	return len(record) > 6 && record[:6] == "v=oasf"
}

// truncateString truncates a string to the given length, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
