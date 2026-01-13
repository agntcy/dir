// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"strings"

	"github.com/agntcy/dir/utils/logging"
)

var domainLogger = logging.Logger("verification/domain")

// ExtractDomain extracts the domain from an OASF record name.
//
// Expected format: <domain>[/<path>]
// Examples:
//   - "cisco.com" -> "cisco.com"
//   - "cisco.com/marketing-agent" -> "cisco.com"
//   - "example.org/agents/v1" -> "example.org"
//
// Returns empty string if the domain is invalid (no dot).
func ExtractDomain(name string) string {
	if name == "" {
		return ""
	}

	// Get domain part (everything before the first slash, or the whole string)
	domain := name
	if idx := strings.Index(name, "/"); idx != -1 {
		domain = name[:idx]
	}

	// Basic validation: must contain at least one dot
	if !strings.Contains(domain, ".") {
		domainLogger.Debug("Invalid domain: no dot found", "domain", domain)

		return ""
	}

	return domain
}
