// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package dns provides DNS TXT record verification for name ownership.
package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/agntcy/dir/server/naming"
	"github.com/agntcy/dir/server/naming/dns/config"
	"github.com/agntcy/dir/utils/logging"
)

const (
	// DNSRecordPrefix is the subdomain prefix for DIR naming system DNS TXT records.
	DNSRecordPrefix = "_dir_nsys."

	// RecordSchemaPrefix is the prefix that identifies DIR naming system TXT records.
	RecordSchemaPrefix = "schema=v1"

	// maxTruncateLen is the maximum length for truncated log strings.
	maxTruncateLen = 50
)

var logger = logging.Logger("naming/dns")

// Resolver handles DNS TXT record lookups for name verification.
type Resolver struct {
	// resolver is the underlying DNS resolver (nil uses default).
	resolver *net.Resolver

	// timeout is the maximum time to wait for DNS resolution.
	timeout time.Duration
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithTimeout sets the DNS resolution timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(r *Resolver) {
		r.timeout = timeout
	}
}

// WithResolver sets a custom DNS resolver.
func WithResolver(resolver *net.Resolver) Option {
	return func(r *Resolver) {
		r.resolver = resolver
	}
}

// NewResolver creates a new DNS resolver with the given options.
func NewResolver(opts ...Option) *Resolver {
	r := &Resolver{
		timeout: config.DefaultTimeout,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// NewResolverFromConfig creates a new DNS resolver from configuration.
func NewResolverFromConfig(cfg *config.Config) *Resolver {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	return NewResolver(WithTimeout(cfg.Timeout))
}

// LookupKeys retrieves public keys from DNS TXT records for the given domain.
// It looks up _dir_nsys.<domain> and parses DIR naming system formatted TXT records.
func (r *Resolver) LookupKeys(ctx context.Context, domain string) ([]naming.PublicKey, error) {
	// Create context with timeout
	lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Build the DNS name to lookup
	dnsName := DNSRecordPrefix + domain

	logger.Debug("Looking up DNS TXT records", "domain", domain, "dnsName", dnsName)

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
				logger.Debug("No DNS TXT records found", "domain", domain)

				return nil, nil // Not an error, just no records
			}
		}

		return nil, fmt.Errorf("DNS lookup failed for %s: %w", dnsName, err)
	}

	logger.Debug("Found DNS TXT records", "domain", domain, "count", len(records))

	// Parse DIR naming system records
	keys := make([]naming.PublicKey, 0, len(records))

	for _, record := range records {
		// Skip non-DIR naming system records
		if !isDirNsysRecord(record) {
			logger.Debug("Skipping non-DIR naming system TXT record", "record", truncateString(record, maxTruncateLen))

			continue
		}

		key, err := ParseTXTRecord(record)
		if err != nil {
			logger.Warn("Failed to parse DIR naming system TXT record", "record", truncateString(record, maxTruncateLen), "error", err)

			continue
		}

		keys = append(keys, *key)
		logger.Debug("Parsed public key from DNS", "domain", domain, "keyType", key.Type)
	}

	return keys, nil
}

// isDirNsysRecord checks if a TXT record is a DIR naming system record.
func isDirNsysRecord(record string) bool {
	// DIR naming system records start with "schema=v1"
	return strings.HasPrefix(record, RecordSchemaPrefix)
}

// truncateString truncates a string to the given length, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
