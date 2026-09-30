// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
)

// TXTPrefix is prepended to the domain to form the key record's DNS name,
// e.g. "_agntcy-key.acme.com".
const TXTPrefix = "_agntcy-key."

// DNS resolves "dns:" subjects (and bare domains) from TXT records of the
// form "v=akv1;key=<base64 DER SubjectPublicKeyInfo>".
type DNS struct {
	lookupTXT func(ctx context.Context, name string) ([]string, error)
}

// NewDNS creates a DNS resolver backed by the system resolver.
func NewDNS() *DNS {
	return &DNS{lookupTXT: net.DefaultResolver.LookupTXT}
}

// Resolve implements Resolver. It ignores certificate.
func (d *DNS) Resolve(ctx context.Context, subject string, _ []byte) ([]crypto.PublicKey, error) {
	u, err := parseHost(strings.TrimPrefix(subject, "dns:"))
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

	return finish(keys, name)
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

	return toPublicKey(pub)
}
