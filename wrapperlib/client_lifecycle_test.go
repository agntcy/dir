// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		addr, cleanup := startTestServer(t)
		defer cleanup()

		reqJSON := fmt.Sprintf(`{"server_address":%q,"auth_mode":"insecure"}`, addr)
		resp := bridgeNoHandle(NewClient, reqJSON)

		var out struct {
			Handle int64  `json:"handle"`
			Error  string `json:"error"`
		}
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if out.Handle == 0 {
			t.Fatal("expected a non-zero handle")
		}

		closeResp := bridgeHandleOnly(CloseClient, out.Handle)

		var closeOut errorResponse
		if err := json.Unmarshal([]byte(closeResp), &closeOut); err != nil {
			t.Fatalf("close response is not valid JSON: %v", err)
		}

		if closeOut.Error != "" {
			t.Fatalf("unexpected error closing client: %s", closeOut.Error)
		}
	})

	t.Run("invalid config JSON", func(t *testing.T) {
		resp := bridgeNoHandle(NewClient, `{not valid json`)

		var out struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if out.Error == "" {
			t.Fatal("expected a non-empty error for malformed config JSON")
		}
	})

	t.Run("unsupported auth mode", func(t *testing.T) {
		resp := bridgeNoHandle(NewClient, `{"server_address":"127.0.0.1:0","auth_mode":"bogus-mode"}`)

		var out struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if out.Error == "" {
			t.Fatal("expected a non-empty error for an unsupported auth mode")
		}
	})
}

func TestCloseClient(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		addr, cleanup := startTestServer(t)
		defer cleanup()

		handle := newTestClientHandle(t, addr)

		resp := bridgeHandleOnly(CloseClient, handle)

		var out errorResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}
	})

	t.Run("unknown handle", func(t *testing.T) {
		resp := bridgeHandleOnly(CloseClient, 999999)

		var out errorResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if out.Error == "" {
			t.Fatal("expected a non-empty error for an unknown handle")
		}
	})

	t.Run("double close surfaces unknown handle", func(t *testing.T) {
		addr, cleanup := startTestServer(t)
		defer cleanup()

		handle := newTestClientHandle(t, addr)

		first := bridgeHandleOnly(CloseClient, handle)

		var firstOut errorResponse
		if err := json.Unmarshal([]byte(first), &firstOut); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if firstOut.Error != "" {
			t.Fatalf("unexpected error on first close: %s", firstOut.Error)
		}

		second := bridgeHandleOnly(CloseClient, handle)

		var secondOut errorResponse
		if err := json.Unmarshal([]byte(second), &secondOut); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if secondOut.Error == "" {
			t.Fatal("expected a non-empty error on second close of the same handle")
		}
	})
}
