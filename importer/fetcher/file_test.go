// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileFetcher_Fetch_bareServerJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "one.json")

	json := `{
  "$schema": "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json",
  "name": "io.example/test",
  "description": "Test server for unit tests",
  "version": "1.0.0",
  "remotes": [{"type": "streamable-http", "url": "https://example.com/mcp"}]
}`
	if err := os.WriteFile(path, []byte(json), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := NewFileFetcher(path)
	if err != nil {
		t.Fatal(err)
	}

	outCh, errCh := f.Fetch(context.Background())

	var got int

	for range outCh {
		got++
	}

	for err := range errCh {
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	}

	if got != 1 {
		t.Fatalf("got %d servers, want 1", got)
	}
}
