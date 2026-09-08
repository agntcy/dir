// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pkgver_test

import (
	"testing"

	"github.com/agntcy/dir/cli/util/pkgver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCanonical is the truth table for what this CLI can order. The short forms
// "1" and "1.0" canonicalize rather than fail, which is deliberate: a record
// tagged "1.0" is still orderable. Everything in the second group carries no
// ordering at all and must never be auto-upgraded across.
func TestCanonical(t *testing.T) {
	orderable := map[string]string{
		"1":              "v1.0.0",
		"v1":             "v1.0.0",
		"1.0":            "v1.0.0",
		"v1.0":           "v1.0.0",
		"1.0.0":          "v1.0.0",
		"v1.0.0":         "v1.0.0",
		"0.0.0":          "v0.0.0",
		"1.10.0":         "v1.10.0",
		"1.0.0-rc.1":     "v1.0.0-rc.1",
		"v1.0.0-rc.1":    "v1.0.0-rc.1",
		"1.0.0+build.5":  "v1.0.0",
		"v1.0.0+build.5": "v1.0.0",
	}

	for raw, want := range orderable {
		assert.Equal(t, want, pkgver.Canonical(raw), "Canonical(%q)", raw)
	}

	unorderable := []string{
		"",
		"main",
		"latest",
		"2026-08-14",
		"1.0.0.0",
		"V1.0.0",
		" v1.0.0",
		"v1.0.0 ",
		"v",
		"1.0.0-",
	}

	for _, raw := range unorderable {
		assert.Empty(t, pkgver.Canonical(raw), "Canonical(%q)", raw)
	}
}

// TestCompareOrdersNumerically is the whole reason this package exists: the
// server's text comparison puts 1.10.0 below 1.9.0.
func TestCompareOrdersNumerically(t *testing.T) {
	assert.Positive(t, pkgver.Compare("1.10.0", "1.9.0"))
	assert.Negative(t, pkgver.Compare("1.9.0", "1.10.0"))
	assert.Positive(t, pkgver.Compare("2.0.0", "1.99.99"))
}

func TestCompareEqualForms(t *testing.T) {
	// Missing "v", short forms, and build metadata are all the same version.
	assert.Equal(t, 0, pkgver.Compare("1.0.0", "v1.0.0"))
	assert.Equal(t, 0, pkgver.Compare("v1", "1.0.0"))
	assert.Equal(t, 0, pkgver.Compare("1.0", "v1.0.0"))
	assert.Equal(t, 0, pkgver.Compare("1.0.0+a", "1.0.0+b"))
}

func TestComparePrereleaseSortsBelowRelease(t *testing.T) {
	assert.Negative(t, pkgver.Compare("1.0.0-rc.1", "1.0.0"))
	assert.Negative(t, pkgver.Compare("1.0.0-rc.1", "1.0.0-rc.2"))
}

func TestCompareIncomparable(t *testing.T) {
	// An unorderable version sorts below every real one, in both directions.
	assert.Negative(t, pkgver.Compare("main", "1.0.0"))
	assert.Positive(t, pkgver.Compare("1.0.0", "main"))
	// Two unorderable versions are equal, so a caller checking for a positive
	// result never treats one as an upgrade over the other.
	assert.Equal(t, 0, pkgver.Compare("main", "latest"))
	assert.Equal(t, 0, pkgver.Compare("", ""))
}

func TestIsPrerelease(t *testing.T) {
	assert.True(t, pkgver.IsPrerelease("1.0.0-rc.1"))
	assert.True(t, pkgver.IsPrerelease("v2.0.0-alpha"))
	assert.False(t, pkgver.IsPrerelease("1.0.0"))
	assert.False(t, pkgver.IsPrerelease("1.0.0+build.5"))
	// Unorderable is not a prerelease; it is simply unordered.
	assert.False(t, pkgver.IsPrerelease("main"))
	assert.False(t, pkgver.IsPrerelease(""))
}

func TestNewest(t *testing.T) {
	got, ok := pkgver.Newest([]string{"1.9.0", "1.10.0", "1.2.0"})
	require.True(t, ok)
	// The caller's spelling is preserved, not the canonical form.
	assert.Equal(t, "1.10.0", got)
}

func TestNewestSkipsIncomparable(t *testing.T) {
	got, ok := pkgver.Newest([]string{"main", "1.0.0", "latest"})
	require.True(t, ok)
	assert.Equal(t, "1.0.0", got)
}

func TestNewestKeepsFirstOfEqualVersions(t *testing.T) {
	got, ok := pkgver.Newest([]string{"v1.0.0", "1.0.0", "1"})
	require.True(t, ok)
	assert.Equal(t, "v1.0.0", got)
}

func TestNewestFindsNothingComparable(t *testing.T) {
	got, ok := pkgver.Newest([]string{"main", "latest", ""})
	assert.False(t, ok)
	assert.Empty(t, got)

	got, ok = pkgver.Newest(nil)
	assert.False(t, ok)
	assert.Empty(t, got)
}
