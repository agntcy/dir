// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"testing"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
)

func TestGetWorkload(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(GetWorkload, handle, `{"workload_id":"workload-1"}`)

		var out workloadResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var workload runtimev1.Workload
		mustProtoUnmarshal(t, out.Workload, &workload)

		if workload.GetId() != "workload-1" {
			t.Fatalf("got id %q, want workload-1", workload.GetId())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(GetWorkload, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(GetWorkload, 999999, `{"workload_id":"workload-1"}`))
	})
}

func TestListWorkloads(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(ListWorkloads, handle, `{"labels":{"env":"prod"}}`)

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
		assertErrorNonEmpty(t, bridgeHandle(ListWorkloads, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(ListWorkloads, 999999, `{}`))
	})
}
