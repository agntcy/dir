// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
)

func TestPublish(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := mustProtoJSON(t, &routingv1.PublishRequest{
			Request: &routingv1.PublishRequest_RecordRefs{
				RecordRefs: &routingv1.RecordRefs{Refs: []*corev1.RecordRef{{Cid: "baecid123"}}},
			},
		})
		assertNoError(t, bridgeHandle(Publish, handle, req))
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Publish, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Publish, 999999, `{}`))
	})
}

func TestUnpublish(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := mustProtoJSON(t, &routingv1.UnpublishRequest{
			Request: &routingv1.UnpublishRequest_RecordRefs{
				RecordRefs: &routingv1.RecordRefs{Refs: []*corev1.RecordRef{{Cid: "baecid123"}}},
			},
		})
		assertNoError(t, bridgeHandle(Unpublish, handle, req))
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Unpublish, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Unpublish, 999999, `{}`))
	})
}

func TestList(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(List, handle, `{}`)

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
		assertErrorNonEmpty(t, bridgeHandle(List, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(List, 999999, `{}`))
	})
}

func TestSearchRouting(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(SearchRouting, handle, `{}`)

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
		assertErrorNonEmpty(t, bridgeHandle(SearchRouting, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(SearchRouting, 999999, `{}`))
	})
}
