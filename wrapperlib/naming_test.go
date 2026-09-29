// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"testing"

	namingv1 "github.com/agntcy/dir/api/naming/v1"
)

func TestGetVerificationInfo(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(GetVerificationInfo, handle, `{"cid":"baecid123"}`)

		var out verificationInfoResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var info namingv1.GetVerificationInfoResponse
		mustProtoUnmarshal(t, out.Info, &info)

		if !info.GetVerified() {
			t.Fatal("expected verified=true")
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(GetVerificationInfo, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(GetVerificationInfo, 999999, `{"cid":"baecid123"}`))
	})
}

func TestGetVerificationInfoByName(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(GetVerificationInfoByName, handle, `{"name":"my-agent"}`)

		var out verificationInfoResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var info namingv1.GetVerificationInfoResponse
		mustProtoUnmarshal(t, out.Info, &info)

		if !info.GetVerified() {
			t.Fatal("expected verified=true")
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(GetVerificationInfoByName, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(GetVerificationInfoByName, 999999, `{"name":"my-agent"}`))
	})
}

func TestResolve(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		resp := bridgeHandle(Resolve, handle, `{"name":"my-agent"}`)

		var out resolveResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var result namingv1.ResolveResponse
		mustProtoUnmarshal(t, out.Result, &result)

		if len(result.GetRecords()) != 1 || result.GetRecords()[0].GetName() != "my-agent" {
			t.Fatalf("unexpected records: %+v", result.GetRecords())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Resolve, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Resolve, 999999, `{"name":"my-agent"}`))
	})
}
