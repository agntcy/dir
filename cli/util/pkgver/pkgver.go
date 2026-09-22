// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package pkgver compares record versions as semantic versions.
//
// Every client-side "is this newer?" decision goes through here, because the
// Directory server compares versions as SQL text: `version > ?` against a text
// column puts "v1.10.0" below "v1.9.0". Ordering therefore has to happen in the
// CLI, on version strings the server hands back.
//
// Comparison is delegated to golang.org/x/mod/semver, which is
// Go-module-flavored in two ways that matter:
//
//   - It requires the "v" prefix. Records commonly store "1.0.0", so Canonical
//     adds one before comparing.
//   - It accepts the short forms "v1" and "v1.0", canonicalizing both to
//     "v1.0.0". That leniency is wanted: a record tagged "1.0" is still
//     orderable against "1.0.1".
//
// Only strings the library rejects outright are incomparable — "main",
// "latest", "2026-08-14", "1.0.0.0", "V1.0.0", the empty string. Canonical
// reports "" for those, and a caller must never auto-upgrade across one:
// nothing can be said about whether it is older or newer than anything else.
package pkgver

import "golang.org/x/mod/semver"

// Canonical returns raw in canonical semver form ("v1.2.3", "v1.2.3-rc.1"), or
// "" when raw carries no ordering. A missing "v" prefix is added. Build
// metadata is dropped, because semver gives it no ordering.
//
// An empty result is the comparability test: Canonical(raw) == "" means no
// claim can be made about where raw sits relative to another version.
func Canonical(raw string) string {
	v := raw
	if v != "" && v[0] != 'v' {
		v = "v" + v
	}

	if !semver.IsValid(v) {
		return ""
	}

	return semver.Canonical(v)
}

// Compare returns -1, 0 or +1 as a sorts before, with, or after b.
//
// Incomparable versions sort below every comparable one, and two of them
// compare equal. That ordering is a convenience for sorting a mixed list, not a
// license to act: before offering an upgrade, check Canonical on both sides, so
// an unorderable installed version is never silently moved.
func Compare(a, b string) int {
	ca, cb := Canonical(a), Canonical(b)

	switch {
	case ca == "" && cb == "":
		return 0
	case ca == "":
		return -1
	case cb == "":
		return 1
	default:
		return semver.Compare(ca, cb)
	}
}

// IsPrerelease reports whether raw is a comparable version carrying a
// prerelease suffix, as "v1.0.0-rc.1" does. An incomparable version is not a
// prerelease — it is simply unordered.
func IsPrerelease(raw string) bool {
	c := Canonical(raw)
	if c == "" {
		return false
	}

	return semver.Prerelease(c) != ""
}

// Newest returns the highest version in versions, spelled the way the caller
// wrote it, and whether one was found. Incomparable entries are ignored and
// ties keep the entry seen first.
//
// A list with nothing comparable in it reports false, so callers fall back to
// their own rule rather than pick arbitrarily.
func Newest(versions []string) (string, bool) {
	best := ""
	found := false

	for _, v := range versions {
		if Canonical(v) == "" {
			continue
		}

		if !found || Compare(v, best) > 0 {
			best = v
			found = true
		}
	}

	return best, found
}
