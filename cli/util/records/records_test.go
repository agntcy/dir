// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package records

import (
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rec(name, version string) *corev1.Record {
	return corev1.New(&oasfv1alpha1.Record{Name: name, Version: version})
}

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "cisco.com-my-agent", SanitizeName("cisco.com/my-agent"))
	assert.Equal(t, "a-b-c", SanitizeName("a:b c"))
}

func TestLatestByNameKeepsHighestSemver(t *testing.T) {
	records := []*corev1.Record{
		rec("agent-a", "1.0.0"),
		rec("agent-a", "2.1.0"),
		rec("agent-b", "0.1.0"),
		rec("agent-a", "1.5.0"),
	}

	latest := LatestByName(records)
	require.Len(t, latest, 2)

	byName := map[string]string{}
	for _, r := range latest {
		byName[r.GetName()] = r.GetVersion()
	}

	assert.Equal(t, "2.1.0", byName["agent-a"])
	assert.Equal(t, "0.1.0", byName["agent-b"])
}

func TestLatestByNameHandlesVPrefixAndPrerelease(t *testing.T) {
	// v-prefixed and bare versions compare correctly.
	vp := LatestByName([]*corev1.Record{rec("x", "v1.0.0"), rec("x", "v2.0.0")})
	require.Len(t, vp, 1)
	assert.Equal(t, "v2.0.0", vp[0].GetVersion())

	// A release beats its prerelease.
	pre := LatestByName([]*corev1.Record{rec("x", "1.0.0-alpha"), rec("x", "1.0.0")})
	require.Len(t, pre, 1)
	assert.NotContains(t, pre[0].GetVersion(), "alpha")
}

func TestLatestByNamePreservesFirstSeenOrderAndEmpty(t *testing.T) {
	out := LatestByName([]*corev1.Record{rec("b", "1.0.0"), rec("a", "1.0.0"), rec("b", "1.0.0")})
	require.Len(t, out, 2)
	assert.Equal(t, "b", out[0].GetName())
	assert.Equal(t, "a", out[1].GetName())

	assert.Empty(t, LatestByName(nil))
}

func TestBatchFileNameUsesNameAndVersion(t *testing.T) {
	seen := map[string]int{}

	assert.Equal(t, "my-agent", BatchFileName(rec("my-agent", "1.0.0"), 0, seen, false))
	assert.Equal(t, "my-agent-1.0.0", BatchFileName(rec("my-agent", "1.0.0"), 0, seen, true))
}

func TestBatchFileNameFallsBackToIndex(t *testing.T) {
	seen := map[string]int{}
	assert.Equal(t, "record_3", BatchFileName(rec("", ""), 3, seen, false))
}

func TestBatchFileNameDeduplicatesCollisions(t *testing.T) {
	seen := map[string]int{}

	first := BatchFileName(rec("dup", "1.0.0"), 0, seen, true)
	second := BatchFileName(rec("dup", "1.0.0"), 1, seen, true)

	assert.Equal(t, "dup-1.0.0", first)
	assert.Equal(t, "dup-1.0.0-1", second)
}
