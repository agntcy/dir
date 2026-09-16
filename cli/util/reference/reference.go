// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package reference provides utilities for parsing Docker-style record references.
package reference

import (
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/util/pkgver"
	"github.com/agntcy/dir/client"
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

// Highest picks the record a name resolves to when the reference names no
// version: the highest semantic version published under it.
//
// The naming service orders its answer by created_at descending, so the first
// record is the newest *push*, which is not the newest version. Push v1.9.0
// after v2.0.0 and the first record is v1.9.0 — so a bare `dirctl install
// <name>` would install v1.9.0 while `dirctl install outdated` calls v2.0.0 the
// latest, and would then report that install as up to date against a version it
// would never have chosen.
//
// Releases beat prereleases, so publishing v2.0.0-rc.1 does not change what a
// bare name means. A name carrying nothing but prereleases resolves to the
// highest of those, since the alternative is not resolving at all.
//
// Versions semver cannot order — "latest", "dev", a date — keep the server's
// order, which is what lets a name tagged that way go on working. A list
// mixing orderable and unorderable versions resolves to the highest orderable
// one, because an unordered tag cannot be shown to be newer than anything.
func Highest(records []*corev1.NamedRecordRef) *corev1.NamedRecordRef {
	if len(records) == 0 {
		return nil
	}

	if best := highestOf(records, false); best != nil {
		return best
	}

	if best := highestOf(records, true); best != nil {
		return best
	}

	return records[0]
}

// highestOf returns the highest orderable version among records, restricted to
// prereleases when pre is set and to releases otherwise.
//
// Ties keep the record seen first, which in the server's newest-pushed-first
// order is the most recent push of a re-pushed version.
func highestOf(records []*corev1.NamedRecordRef, pre bool) *corev1.NamedRecordRef {
	var best *corev1.NamedRecordRef

	for _, rec := range records {
		version := rec.GetVersion()
		if pkgver.Canonical(version) == "" || pkgver.IsPrerelease(version) != pre {
			continue
		}

		if best == nil || pkgver.Compare(version, best.GetVersion()) > 0 {
			best = rec
		}
	}

	return best
}

// ResolveToCID resolves the input to a CID using the provided client.
// Supports formats:
//   - CID directly (e.g., "bafyreib...")
//   - name (e.g., "cisco.com/agent") -> highest version, see Highest
//   - name:version (e.g., "cisco.com/agent:v1.0.0") -> specific version
//   - name@digest (e.g., "cisco.com/agent@bafyreib...") -> hash-verified lookup
//   - name:version@digest -> hash-verified lookup of specific version
func ResolveToCID(ctx context.Context, c *client.Client, input string) (string, error) {
	// Parse the input as a reference
	ref := Parse(input)

	// If it's a raw CID (no name), use it directly
	if ref.IsCID() {
		return ref.Digest, nil
	}

	// Resolve the name via the server
	resp, err := c.Resolve(ctx, ref.Name, ref.Version)
	if err != nil {
		return "", fmt.Errorf("failed to resolve record: %w", err)
	}

	records := resp.GetRecords()
	if len(records) == 0 {
		return "", fmt.Errorf("no records found for %q", input)
	}

	// A reference naming an explicit version is already narrowed to it by the
	// server, so Highest only ever chooses between re-pushes of that one
	// version, where it keeps the server's newest-first order.
	resolvedCID := Highest(records).GetCid()

	// If a digest was provided, verify it matches the resolved CID
	if ref.HasDigest() && ref.Digest != resolvedCID {
		return "", fmt.Errorf("hash verification failed: resolved CID %q does not match expected digest %q",
			resolvedCID, ref.Digest)
	}

	return resolvedCID, nil
}
