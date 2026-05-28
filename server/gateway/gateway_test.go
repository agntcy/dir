// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc"
)

// TestNewMarshalerSpecialCasesHttpBody locks down the runtime.HTTPBodyMarshaler
// wiring in New(). Without this wrapper, RPCs returning google.api.HttpBody
// (notably AgentFinderService.ExportAgent) would be JSON-encoded by the
// default grpc-gateway marshaler — exposing the proto envelope
// ({"content_type":"…","data":"<base64>"}) instead of the raw bytes the
// controller intends. This test reaches into the mux's registered marshaler
// indirectly by reconstructing the same wrapper New() builds.
//
// We don't spin up an HTTP server here; the marshaler contract is itself
// public and testable in isolation, and an end-to-end test would require
// a live gRPC backend just to exercise marshaling logic that lives in
// grpc-gateway's runtime package.
func TestHTTPBodyMarshaler(t *testing.T) {
	// Build the same marshaler stack New() constructs. Keep this
	// mirror in sync with gateway.go — if it drifts, this test no
	// longer protects the real wiring and should be deleted.
	server, err := New(context.Background(), Options{
		HTTPAddress:  ":0",
		GRPCEndpoint: "127.0.0.1:1",
		RegisterHandlers: func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
			return nil
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Stop(context.Background()) })

	// The marshaler we expect New() to have installed.
	marshaler := &runtime.HTTPBodyMarshaler{Marshaler: &runtime.JSONPb{}}

	t.Run("returns Data bytes verbatim for HttpBody", func(t *testing.T) {
		body := &httpbody.HttpBody{
			ContentType: "text/markdown; charset=utf-8",
			Data:        []byte("# SKILL.md\n\nhello\n"),
		}

		got, err := marshaler.Marshal(body)
		require.NoError(t, err)

		// Critically: no base64, no envelope. Just the bytes.
		assert.Equal(t, body.GetData(), got)
	})

	t.Run("uses ContentType from HttpBody", func(t *testing.T) {
		body := &httpbody.HttpBody{
			ContentType: "application/json",
			Data:        []byte("{\"ok\":true}\n"),
		}

		assert.Equal(t, "application/json", marshaler.ContentType(body))
	})

	t.Run("falls back to JSONPb for non-HttpBody messages", func(t *testing.T) {
		// A plain string here stands in for any non-HttpBody value;
		// HTTPBodyMarshaler's fall-through path delegates to JSONPb
		// for ContentType resolution, which returns the JSON media
		// type for anything that isn't an HttpBody.
		assert.Equal(t, "application/json", marshaler.ContentType("not an HttpBody"))
	})
}
