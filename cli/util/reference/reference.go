// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package reference provides utilities for parsing Docker-style record references.
package reference

import (
	"strings"

	"github.com/ipfs/go-cid"
	"golang.org/x/mod/semver"
)

// Ref represents a parsed record reference.
type Ref struct {
	// Name is the record name (e.g., "cisco.com/agent").
	Name string

	// Version is the optional version (e.g., "v1.0.0").
	Version string

	// CID is set if the input is a raw CID (direct pull by content address).
	CID string
}

// IsCID returns true if the reference is just a raw CID.
func (r Ref) IsCID() bool {
	return r.CID != ""
}

// Parse parses an input string into a record reference.
// Supports formats:
//   - CID directly: "bafyreib..." -> CID (direct pull by content address)
//   - Name only: "cisco.com/agent" -> Name (resolves to latest version)
//   - Name:version: "cisco.com/agent:v1.0.0" -> Name + Version (resolves to specific version)
func Parse(input string) Ref {
	// If the entire input is a valid CID, return it as a CID reference
	if IsCID(input) {
		return Ref{CID: input}
	}

	// Parse name:version
	name, version := parseNameAndVersion(input)

	return Ref{Name: name, Version: version}
}

// parseNameAndVersion parses input in the format "name:version" or just "name".
func parseNameAndVersion(input string) (string, string) {
	// Find the last colon that could be a version separator
	// We need to be careful with URLs like "https://example.com:8080/agent:v1.0.0"
	// The version is typically after the last colon and starts with 'v' or a digit
	lastColon := strings.LastIndex(input, ":")
	if lastColon == -1 {
		return input, ""
	}

	possibleVersion := input[lastColon+1:]

	// Check if it looks like a version (starts with v or digit)
	if len(possibleVersion) > 0 && (possibleVersion[0] == 'v' || (possibleVersion[0] >= '0' && possibleVersion[0] <= '9')) {
		// Make sure it's not a port number in a URL (e.g., localhost:8080)
		// Ports are typically followed by / or nothing, versions have dots
		if strings.Contains(possibleVersion, ".") || semver.IsValid("v"+possibleVersion) || semver.IsValid(possibleVersion) {
			return input[:lastColon], possibleVersion
		}
	}

	return input, ""
}

// IsCID checks if the input string is a valid CID.
func IsCID(input string) bool {
	_, err := cid.Decode(input)

	return err == nil
}
