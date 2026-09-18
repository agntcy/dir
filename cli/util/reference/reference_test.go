// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reference_test

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ref builds one naming-service answer row. The CID is derived from the version
// so an assertion can name the version it expects rather than a digest.
func ref(version string) *corev1.NamedRecordRef {
	return &corev1.NamedRecordRef{Name: "cisco.com/agent", Version: version, Cid: "cid-" + version}
}

// TestHighestIgnoresPushOrder is the whole reason Highest exists: the naming
// service answers newest-pushed first, so a v1.9.0 pushed after v2.0.0 comes
// back first and used to be what a bare name resolved to.
func TestHighestIgnoresPushOrder(t *testing.T) {
	t.Parallel()

	got := reference.Highest([]*corev1.NamedRecordRef{ref("1.9.0"), ref("2.0.0")})
	require.NotNil(t, got)
	assert.Equal(t, "cid-2.0.0", got.GetCid())
}

// TestHighestOrdersNumerically guards the same trap the server's text
// comparison falls into: "1.10.0" sorts below "1.9.0" as text.
func TestHighestOrdersNumerically(t *testing.T) {
	t.Parallel()

	got := reference.Highest([]*corev1.NamedRecordRef{ref("1.9.0"), ref("1.10.0")})
	require.NotNil(t, got)
	assert.Equal(t, "cid-1.10.0", got.GetCid())
}

// TestHighestPrefersReleaseOverPrerelease: publishing a release candidate must
// not change what a bare name means.
func TestHighestPrefersReleaseOverPrerelease(t *testing.T) {
	t.Parallel()

	got := reference.Highest([]*corev1.NamedRecordRef{ref("2.0.0-rc.1"), ref("1.0.0")})
	require.NotNil(t, got)
	assert.Equal(t, "cid-1.0.0", got.GetCid())
}

// TestHighestFallsBackToPrereleases: a name with nothing but prereleases has to
// resolve to something, so the highest prerelease wins.
func TestHighestFallsBackToPrereleases(t *testing.T) {
	t.Parallel()

	got := reference.Highest([]*corev1.NamedRecordRef{ref("1.0.0-rc.1"), ref("1.0.0-rc.2")})
	require.NotNil(t, got)
	assert.Equal(t, "cid-1.0.0-rc.2", got.GetCid())
}

// TestHighestKeepsServerOrderForUnorderableVersions: a name tagged "latest" or
// "dev" carries no ordering, so the server's newest-pushed-first order stands.
func TestHighestKeepsServerOrderForUnorderableVersions(t *testing.T) {
	t.Parallel()

	got := reference.Highest([]*corev1.NamedRecordRef{ref("dev"), ref("latest")})
	require.NotNil(t, got)
	assert.Equal(t, "cid-dev", got.GetCid())
}

// TestHighestSkipsUnorderableVersionsWhenOneIsOrderable: an unordered tag
// cannot be shown to be newer than a real version, so it never wins over one.
func TestHighestSkipsUnorderableVersionsWhenOneIsOrderable(t *testing.T) {
	t.Parallel()

	got := reference.Highest([]*corev1.NamedRecordRef{ref("latest"), ref("1.0.0")})
	require.NotNil(t, got)
	assert.Equal(t, "cid-1.0.0", got.GetCid())
}

// TestHighestKeepsNewestPushOfARepushedVersion: two rows on one version are a
// re-push, and the server's order puts the newest first.
func TestHighestKeepsNewestPushOfARepushedVersion(t *testing.T) {
	t.Parallel()

	newest := &corev1.NamedRecordRef{Version: "1.0.0", Cid: "cid-new"}
	older := &corev1.NamedRecordRef{Version: "v1.0.0", Cid: "cid-old"}

	got := reference.Highest([]*corev1.NamedRecordRef{newest, older})
	require.NotNil(t, got)
	assert.Equal(t, "cid-new", got.GetCid())
}

func TestHighestOnEmptyInput(t *testing.T) {
	t.Parallel()

	assert.Nil(t, reference.Highest(nil))
	assert.Nil(t, reference.Highest([]*corev1.NamedRecordRef{}))
}

// TestParseKeepsVersionAndDigestApart covers the reference grammar Highest sits
// behind: only a version-less reference reaches the version choice at all.
func TestParseKeepsVersionAndDigestApart(t *testing.T) {
	t.Parallel()

	bare := reference.Parse("cisco.com/agent")
	assert.Equal(t, "cisco.com/agent", bare.Name)
	assert.Empty(t, bare.Version)
	assert.False(t, bare.HasDigest())

	versioned := reference.Parse("cisco.com/agent:v1.0.0")
	assert.Equal(t, "cisco.com/agent", versioned.Name)
	assert.Equal(t, "v1.0.0", versioned.Version)

	digested := reference.Parse("cisco.com/agent:v1.0.0@sha256:abc")
	assert.Equal(t, "cisco.com/agent", digested.Name)
	assert.Equal(t, "v1.0.0", digested.Version)
	assert.Equal(t, "sha256:abc", digested.Digest)
}
