// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"errors"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc"
)

func noopRegister(context.Context, *runtime.ServeMux, *grpc.ClientConn) error { return nil }

func TestNew_Validation(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{
			name: "missing http address",
			opts: Options{GRPCEndpoint: "127.0.0.1:8888", RegisterHandlers: noopRegister},
		},
		{
			name: "missing grpc endpoint",
			opts: Options{HTTPAddress: ":8889", RegisterHandlers: noopRegister},
		},
		{
			name: "missing register handlers",
			opts: Options{HTTPAddress: ":8889", GRPCEndpoint: "127.0.0.1:8888"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := New(context.Background(), tt.opts)
			require.Error(t, err)
			assert.Nil(t, server)
		})
	}
}

func TestNew_RegistrationFailure(t *testing.T) {
	wantErr := errors.New("boom")

	server, err := New(context.Background(), Options{
		HTTPAddress:  ":0",
		GRPCEndpoint: "127.0.0.1:8888",
		RegisterHandlers: func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
			return wantErr
		},
	})

	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, server)
}

func TestNew_RegisterAIFinder(t *testing.T) {
	// Routes wire up even though the gRPC service is not served yet (lazy conn).
	server, err := New(context.Background(), Options{
		HTTPAddress:      ":0",
		GRPCEndpoint:     "127.0.0.1:8888",
		RegisterHandlers: RegisterAIFinder,
	})
	require.NoError(t, err)
	require.NotNil(t, server)
	t.Cleanup(func() { _ = server.Stop(context.Background()) })
}

func TestStartStop(t *testing.T) {
	server, err := New(context.Background(), Options{
		HTTPAddress:      ":0",
		GRPCEndpoint:     "127.0.0.1:8888",
		RegisterHandlers: noopRegister,
	})
	require.NoError(t, err)

	require.NoError(t, server.Start())
	require.NoError(t, server.Stop(context.Background()))
}

// TestHTTPBodyMarshaler verifies the HttpBody contract New() installs: raw
// bytes verbatim with the supplied Content-Type, JSON fallback otherwise.
func TestHTTPBodyMarshaler(t *testing.T) {
	marshaler := &runtime.HTTPBodyMarshaler{Marshaler: &runtime.JSONPb{}}

	t.Run("returns Data bytes verbatim for HttpBody", func(t *testing.T) {
		body := &httpbody.HttpBody{
			ContentType: "text/markdown; charset=utf-8",
			Data:        []byte("# SKILL.md\n\nhello\n"),
		}

		got, err := marshaler.Marshal(body)
		require.NoError(t, err)
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
		assert.Equal(t, "application/json", marshaler.ContentType("not an HttpBody"))
	})
}
