// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConnectionErrorsNameTheServer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// An HTTP/1.1 server where a Directory server was expected: the connection error
	// is about the server preface and says nothing about where it went.
	httpServer := httptest.NewServer(http.NotFoundHandler())
	defer httpServer.Close()

	address := httpServer.Listener.Addr().String()

	ctx, cancel := context.WithTimeout(context.Background(), testContextTimeout)
	defer cancel()

	c, err := New(ctx, WithConfig(&Config{ServerAddress: address}))
	require.NoError(t, err)

	defer c.Close()

	_, err = c.ListWorkloadsStreamWithRequest(ctx, &runtimev1.ListWorkloadsRequest{})
	require.ErrorContains(t, err, "(server "+address+")")
	require.Equal(t, codes.Unavailable, status.Code(err))

	_, err = c.GetProviderCount(ctx, &routingv1.GetProviderCountRequest{})
	require.ErrorContains(t, err, "(server "+address+")")
	require.Equal(t, codes.Unavailable, status.Code(err))
}

func TestWithServerAddressOnlyChangesUnavailableErrors(t *testing.T) {
	const address = "dir.example.com:443"

	require.NoError(t, withServerAddress(nil, address))
	require.ErrorIs(t, withServerAddress(io.EOF, address), io.EOF)

	notFound := status.Error(codes.NotFound, "record not found")
	require.Equal(t, notFound, withServerAddress(notFound, address))

	unavailable := withServerAddress(status.Error(codes.Unavailable, "connection error"), address)
	require.Equal(t, codes.Unavailable, status.Code(unavailable))
	require.Equal(t, "connection error (server dir.example.com:443)", status.Convert(unavailable).Message())
}
