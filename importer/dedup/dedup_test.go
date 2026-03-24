// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dedup

import (
	"context"
	"testing"

	"github.com/agntcy/dir/importer/types"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

func TestExtractNameVersionFromSource(t *testing.T) {
	t.Parallel()

	srv := mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "n", Version: "v"}}
	if got := extractNameVersionFromSource(srv); got != "n@v" {
		t.Errorf("value = %q, want n@v", got)
	}

	ptr := &mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "a", Version: "b"}}
	if got := extractNameVersionFromSource(ptr); got != "a@b" {
		t.Errorf("ptr = %q, want a@b", got)
	}

	if got := extractNameVersionFromSource(mcpapiv0.ServerResponse{}); got != "" {
		t.Errorf("empty = %q, want \"\"", got)
	}

	if got := extractNameVersionFromSource("not-a-response"); got != "" {
		t.Errorf("wrong type = %q, want \"\"", got)
	}
}

func TestFilterDuplicates_SkipsKnownDuplicate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := &types.Result{}

	c := &MCPDuplicateChecker{
		existingRecords: map[string]string{"dup@1.0.0": "bafycid"},
	}

	in := make(chan mcpapiv0.ServerResponse, 2)
	in <- mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "dup", Version: "1.0.0"}}

	in <- mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "new", Version: "2.0.0"}}

	close(in)

	out := c.FilterDuplicates(ctx, in, result)

	var passed int

	for range out {
		passed++
	}

	if passed != 1 {
		t.Errorf("passed through = %d, want 1", passed)
	}

	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", result.SkippedCount)
	}

	// Dedup increments TotalRecords for duplicates; transform would add for non-dupes (not run here).
	if result.TotalRecords != 1 {
		t.Errorf("TotalRecords after dedup = %d, want 1 (duplicate only)", result.TotalRecords)
	}
}

func TestFilterDuplicates_PassThroughWhenUnknown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := &types.Result{}

	c := &MCPDuplicateChecker{existingRecords: map[string]string{}}

	in := make(chan mcpapiv0.ServerResponse, 1)
	in <- mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "only", Version: "1"}}

	close(in)

	out := c.FilterDuplicates(ctx, in, result)

	n := 0

	for range out {
		n++
	}

	if n != 1 {
		t.Errorf("count = %d, want 1", n)
	}

	if result.SkippedCount != 0 {
		t.Errorf("SkippedCount = %d", result.SkippedCount)
	}
}
