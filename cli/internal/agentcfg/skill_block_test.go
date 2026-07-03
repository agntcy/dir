// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testSlug = "agntcy-dir"

func TestUpsertBlockInsertsIntoEmpty(t *testing.T) {
	out := upsertBlock(testSlug, "", "HELLO")

	assert.True(t, hasBlock(testSlug, out))
	assert.Contains(t, out, blockBegin(testSlug))
	assert.Contains(t, out, "HELLO")
	assert.Contains(t, out, blockEnd(testSlug))
}

func TestUpsertBlockAppendsPreservingUserContent(t *testing.T) {
	existing := "# My rules\n\nKeep this.\n"

	out := upsertBlock(testSlug, existing, "OURS")

	assert.Contains(t, out, "Keep this.")
	assert.Contains(t, out, "OURS")
	assert.Less(t, strings.Index(out, "Keep this."), strings.Index(out, "OURS"))
}

func TestUpsertBlockReplacesExistingBlock(t *testing.T) {
	first := upsertBlock(testSlug, "# Header\n", "OLD CONTENT")
	second := upsertBlock(testSlug, first, "NEW CONTENT")

	assert.Contains(t, second, "NEW CONTENT")
	assert.NotContains(t, second, "OLD CONTENT")
	assert.Contains(t, second, "# Header")
	// Only one block.
	assert.Equal(t, 1, strings.Count(second, blockBegin(testSlug)))
}

func TestRemoveBlockStripsOnlyOurBlock(t *testing.T) {
	existing := "# Header\n\nUser text.\n"
	withBlock := upsertBlock(testSlug, existing, "OURS")

	out, removed := removeBlock(testSlug, withBlock)
	assert.True(t, removed)
	assert.NotContains(t, out, "OURS")
	assert.NotContains(t, out, blockBegin(testSlug))
	assert.Contains(t, out, "User text.")
}

func TestRemoveBlockAbsentReturnsFalse(t *testing.T) {
	out, removed := removeBlock(testSlug, "no block here\n")
	assert.False(t, removed)
	assert.Equal(t, "no block here\n", out)
}

func TestManagedBlockTwoSlugsCoexist(t *testing.T) {
	existing := ""
	existing = upsertBlock("record-a", existing, "body A")
	existing = upsertBlock("record-b", existing, "body B")

	if !hasBlock("record-a", existing) || !hasBlock("record-b", existing) {
		t.Fatalf("both blocks should be present:\n%s", existing)
	}

	stripped, removed := removeBlock("record-a", existing)
	if !removed {
		t.Fatal("record-a block should have been removed")
	}

	if hasBlock("record-a", stripped) {
		t.Fatal("record-a block should be gone")
	}

	if !hasBlock("record-b", stripped) {
		t.Fatalf("record-b block must survive:\n%s", stripped)
	}
}
