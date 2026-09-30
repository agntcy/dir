// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dnsresolver

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	"github.com/agntcy/dir/client/utils/identity/resolvers/internal/keyutil"
)

// TXTPrefix is prepended to the domain to form the key record's DNS name,
// e.g. "_agntcy-key.acme.com".
const TXTPrefix = "_agntcy-key."

// Resolver resolves "dns:" subjects (and bare domains) from TXT records of the
// form "v=akv1;key=<base64 DER SubjectPublicKeyInfo>".
type Resolver struct {
	lookupTXT func(ctx context.Context, name string) ([]string, error)
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithLookupTXT replaces the system DNS lookup, for tests.
func WithLookupTXT(lookup func(ctx context.Context, name string) ([]string, error)) Option {
	return func(r *Resolver) { r.lookupTXT = lookup }
}

// New creates a Resolver backed by the system DNS resolver.
func New(opts ...Option) *Resolver {
	r := &Resolver{lookupTXT: net.DefaultResolver.LookupTXT}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Resolve implements resolvers.Resolver. It ignores certificate.
func (d *Resolver) Resolve(ctx context.Context, subject string, _ []byte) ([]crypto.PublicKey, error) {
	u, err := keyutil.ParseHost(strings.TrimPrefix(subject, "dns:"))
	if err != nil || u.Port() != "" {
		return nil, fmt.Errorf("invalid dns subject %q", subject)
	}

	name := TXTPrefix + u.Hostname()

	records, err := d.lookupTXT(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("lookup TXT records for %s: %w", name, err)
	}

	keys := make([]crypto.PublicKey, 0, len(records))

	for _, record := range records {
		if key, ok := parseKeyRecord(record); ok {
			keys = append(keys, key)
		}
	}

	found, err := keyutil.Finish(keys, name)
	if err != nil {
		return nil, fmt.Errorf("dns: %w", err)
	}

	return found, nil
}

// parseKeyRecord parses one TXT record, skipping any that aren't a
// well-formed "akv1" key record.
func parseKeyRecord(record string) (crypto.PublicKey, bool) {
	var (
		versionOK bool
		keyB64    string
	)

	for field := range strings.SplitSeq(record, ";") {
		name, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}

		switch strings.TrimSpace(name) {
		case "v":
			versionOK = strings.TrimSpace(value) == "akv1"
		case "key":
			keyB64 = strings.TrimSpace(value)
		}
	}

	if !versionOK || keyB64 == "" {
		return nil, false
	}

	der, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, false
	}

	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, false
	}

	return keyutil.ToPublicKey(pub)
}
