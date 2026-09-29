// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"testing"
)

func TestListenStream(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		// The mock server sends 2 canned events and closes the stream, so this
		// call returns promptly instead of hanging (see server_test.go).
		resp := bridgeHandle(ListenStream, handle, `{}`)

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
		assertErrorNonEmpty(t, bridgeHandle(ListenStream, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(ListenStream, 999999, `{}`))
	})
}
