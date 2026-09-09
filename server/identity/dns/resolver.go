// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package dns verifies "dns:" (and bare-domain) subjects against a public
// key published in a DNS TXT record.
package dns

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
)

// TXTPrefix is prepended to the domain to form the key-record's DNS name,
// e.g. "_agntcy-key.acme.com".
const TXTPrefix = "_agntcy-key."

// Resolver verifies claim signatures against a domain's published DNS TXT key record.
type Resolver struct {
	lookupTXT func(ctx context.Context, name string) ([]string, error)
}

// New creates a new DNS TXT resolver using the system resolver.
func New() *Resolver {
	return &Resolver{lookupTXT: net.DefaultResolver.LookupTXT}
}

// Scheme implements identity.Resolver.
func (*Resolver) Scheme() string {
	return "dns"
}

// Verify implements identity.Resolver.
func (r *Resolver) Verify(ctx context.Context, subject string, signature, payload []byte) (bool, error) {
	domain, err := domainFromSubject(subject)
	if err != nil {
		return false, err
	}

	records, err := r.lookupTXT(ctx, TXTPrefix+domain)
	if err != nil {
		return false, fmt.Errorf("lookup TXT records for %s: %w", domain, err)
	}

	for _, record := range records {
		keyDER, ok := parseKeyRecord(record)
		if !ok {
			continue
		}

		pub, err := x509.ParsePKIXPublicKey(keyDER)
		if err != nil {
			continue
		}

		if identityv1.VerifyJWS(string(signature), pub, payload) == nil {
			return true, nil
		}
	}

	return false, nil
}

// domainFromSubject extracts the domain from a "dns:domain" (or bare
// "domain") subject.
func domainFromSubject(subject string) (string, error) {
	domain := strings.TrimPrefix(subject, "dns:")
	if domain == "" {
		return "", fmt.Errorf("invalid dns subject: %q", subject)
	}

	return domain, nil
}

// parseKeyRecord parses a TXT record of the form
// "v=akv1;key=<base64 SPKI DER>", returning the decoded key bytes.
func parseKeyRecord(record string) ([]byte, bool) {
	var (
		hasVersion bool
		keyB64     string
	)

	for _, field := range strings.Split(record, ";") {
		name, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}

		switch strings.TrimSpace(name) {
		case "v":
			hasVersion = strings.TrimSpace(value) == "akv1"
		case "key":
			keyB64 = strings.TrimSpace(value)
		}
	}

	if !hasVersion || keyB64 == "" {
		return nil, false
	}

	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, false
	}

	return key, true
}
