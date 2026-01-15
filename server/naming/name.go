// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"strings"

	"github.com/agntcy/dir/utils/logging"
)

var nameLogger = logging.Logger("naming/name")

// Protocol prefixes for name verification.
const (
	// DNSProtocol indicates DNS TXT record verification.
	DNSProtocol = "dns://"

	// HTTPSProtocol indicates JWKS well-known verification via HTTPS.
	HTTPSProtocol = "https://"

	// HTTPProtocol indicates JWKS well-known verification via HTTP (testing only).
	HTTPProtocol = "http://"
)

// ParsedName represents a parsed record name with optional protocol prefix.
type ParsedName struct {
	// Protocol is the verification protocol (dns://, https://, http://, or empty).
	Protocol string
	// Domain is the domain part of the name.
	Domain string
	// Path is the optional path after the domain.
	Path string
	// FullName is the original name without protocol prefix.
	FullName string
}

// ParseName parses a record name, extracting any protocol prefix.
//
// Expected formats:
//   - "dns://cisco.com/agent" -> Protocol: "dns://", Domain: "cisco.com", Path: "agent"
//   - "https://cisco.com/agent" -> Protocol: "https://", Domain: "cisco.com", Path: "agent"
//   - "http://localhost:8080/agent" -> Protocol: "http://", Domain: "localhost:8080", Path: "agent"
//   - "cisco.com/agent" -> Protocol: "", Domain: "cisco.com", Path: "agent" (no verification)
//   - "cisco.com" -> Protocol: "", Domain: "cisco.com", Path: ""
//
// Returns nil if the name is invalid.
func ParseName(name string) *ParsedName {
	if name == "" {
		return nil
	}

	result := &ParsedName{}

	// Check for protocol prefix
	remaining := name
	switch {
	case strings.HasPrefix(name, DNSProtocol):
		result.Protocol = DNSProtocol
		remaining = strings.TrimPrefix(name, DNSProtocol)
	case strings.HasPrefix(name, HTTPSProtocol):
		result.Protocol = HTTPSProtocol
		remaining = strings.TrimPrefix(name, HTTPSProtocol)
	case strings.HasPrefix(name, HTTPProtocol):
		result.Protocol = HTTPProtocol
		remaining = strings.TrimPrefix(name, HTTPProtocol)
	}

	result.FullName = remaining

	// Split domain and path
	if idx := strings.Index(remaining, "/"); idx != -1 {
		result.Domain = remaining[:idx]
		result.Path = remaining[idx+1:]
	} else {
		result.Domain = remaining
	}

	// Validate domain (must contain at least one dot or be localhost with port)
	if !strings.Contains(result.Domain, ".") && !strings.HasPrefix(result.Domain, "localhost") {
		nameLogger.Debug("Invalid domain: no dot found and not localhost", "domain", result.Domain)

		return nil
	}

	return result
}

// ExtractDomain extracts the domain from a record name.
// This is a convenience function that wraps ParseName.
//
// Returns empty string if the name is invalid.
func ExtractDomain(name string) string {
	parsed := ParseName(name)
	if parsed == nil {
		return ""
	}

	return parsed.Domain
}
