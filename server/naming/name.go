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
	DNSProtocol       = "dns://"
	WellKnownProtocol = "wellknown://"
)

// ParsedName represents a parsed record name with optional protocol prefix.
type ParsedName struct {
	// Protocol is the verification protocol (dns://, wellknown://, or empty for auto).
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
//   - "wellknown://cisco.com/agent" -> Protocol: "wellknown://", Domain: "cisco.com", Path: "agent"
//   - "cisco.com/agent" -> Protocol: "", Domain: "cisco.com", Path: "agent" (auto-detect)
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
	case strings.HasPrefix(name, WellKnownProtocol):
		result.Protocol = WellKnownProtocol
		remaining = strings.TrimPrefix(name, WellKnownProtocol)
	}

	result.FullName = remaining

	// Split domain and path
	if idx := strings.Index(remaining, "/"); idx != -1 {
		result.Domain = remaining[:idx]
		result.Path = remaining[idx+1:]
	} else {
		result.Domain = remaining
	}

	// Validate domain (must contain at least one dot)
	if !strings.Contains(result.Domain, ".") {
		nameLogger.Debug("Invalid domain: no dot found", "domain", result.Domain)

		return nil
	}

	return result
}

// ExtractDomain extracts the domain from an OASF record name.
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
