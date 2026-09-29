// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"testing"

	storev1 "github.com/agntcy/dir/api/store/v1"
)

func TestCreateSync(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(CreateSync, handle, `{"remote_url":"https://remote.example.com","cids":["baecid123"]}`)

		var out createSyncResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if out.SyncID == "" {
			t.Fatal("expected a non-empty sync_id")
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(CreateSync, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(CreateSync, 999999, `{"remote_url":"https://remote.example.com"}`))
	})
}

func TestListSyncs(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(ListSyncs, handle, `{}`)

		var out arrayResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if len(out.Results) != 2 {
			t.Fatalf("got %d results, want 2", len(out.Results))
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(ListSyncs, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(ListSyncs, 999999, `{}`))
	})
}

func TestGetSync(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(GetSync, handle, `{"sync_id":"sync-123"}`)

		var out getSyncResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var sync storev1.GetSyncResponse
		mustProtoUnmarshal(t, out.Sync, &sync)

		if sync.GetSyncId() != "sync-123" {
			t.Fatalf("got sync_id %q, want sync-123", sync.GetSyncId())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(GetSync, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(GetSync, 999999, `{"sync_id":"sync-123"}`))
	})
}

func TestDeleteSync(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		assertNoError(t, bridgeHandle(DeleteSync, handle, `{"sync_id":"sync-123"}`))
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(DeleteSync, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(DeleteSync, 999999, `{"sync_id":"sync-123"}`))
	})
}
