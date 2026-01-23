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

	// Digest is the optional digest/CID for hash-verification (e.g., "bafyreib...").
	// When set with Name, it's used to verify the resolved record matches this CID.
	Digest string
}

// IsCID returns true if the reference is just a raw CID (no name).
func (r Ref) IsCID() bool {
	return r.Digest != "" && r.Name == ""
}

// HasDigest returns true if the reference has a digest for hash-verification.
func (r Ref) HasDigest() bool {
	return r.Digest != ""
}

// Parse parses an input string into a record reference.
// Supports formats:
//   - CID directly: "bafyreib..." -> Digest only (direct pull by content address)
//   - Name only: "cisco.com/agent" -> Name (resolves to latest version)
//   - Name:version: "cisco.com/agent:v1.0.0" -> Name + Version (resolves to specific version)
//   - Name@digest: "cisco.com/agent@bafyreib..." -> Name + Digest (hash-verified pull)
//   - Name:version@digest: "cisco.com/agent:v1.0.0@bafyreib..." -> Name + Version + Digest
func Parse(input string) Ref {
	// If the entire input is a valid CID, return it as a digest-only reference
	if IsCID(input) {
		return Ref{Digest: input}
	}

	ref := Ref{}

	// Check for @digest suffix first - accept any non-empty string after @
	// The server will validate if it's a valid CID
	if atIdx := strings.LastIndex(input, "@"); atIdx != -1 && atIdx < len(input)-1 {
		ref.Digest = input[atIdx+1:]
		input = input[:atIdx]
	}

	// Parse name:version from remaining input
	ref.Name, ref.Version = parseNameAndVersion(input)

	return ref
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
