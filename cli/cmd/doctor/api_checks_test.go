// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNormalizeClientErrorAddsIssuerHintForAmbiguousOIDCCache(t *testing.T) {
	err := normalizeClientError(&client.AmbiguousTokenCacheError{
		Issuers: []string{"https://issuer-a.example.com", "https://issuer-b.example.com"},
	})

	assert.Contains(t, err.Error(), "multiple cached OIDC issuers found")
	assert.Contains(t, err.Error(), "--oidc-issuer")
	assert.Contains(t, err.Error(), "DIRECTORY_CLIENT_OIDC_ISSUER")
}

func TestNormalizeClientErrorAddsLoginHintForMissingOIDCToken(t *testing.T) {
	err := normalizeClientError(assert.AnError)
	assert.Equal(t, assert.AnError, err)

	err = normalizeClientError(errors.New("no OIDC access token: missing"))
	assert.Contains(t, err.Error(), "dirctl auth login")
	assert.Contains(t, err.Error(), "DIRECTORY_CLIENT_AUTH_TOKEN")
}

func TestDirectoryAPI(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer listener.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)

		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
	}()

	result := directoryAPI(context.Background(), listener.Addr().String(), time.Second)

	assert.Equal(t, "directory_api_tcp", result.Name)
	assert.Equal(t, statusPass, result.Status)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("listener did not accept doctor TCP probe")
	}

	require.NoError(t, listener.Close())
	result = directoryAPI(context.Background(), listener.Addr().String(), time.Millisecond)
	assert.Equal(t, statusFail, result.Status)
	assert.Contains(t, result.Details, "error")
}

func TestRoutingList(t *testing.T) {
	tests := []struct {
		name       string
		listErr    error
		recvResp   *routingv1.ListResponse
		recvErr    error
		wantStatus checkStatus
		wantMsg    string
	}{
		{name: "list setup error", listErr: assert.AnError, wantStatus: statusFail, wantMsg: "List RPC failed"},
		{name: "empty stream", recvErr: io.EOF, wantStatus: statusPass, wantMsg: "no local records returned"},
		{name: "recv error", recvErr: assert.AnError, wantStatus: statusFail, wantMsg: "List stream failed"},
		{name: "record returned", recvResp: &routingv1.ListResponse{}, wantStatus: statusPass, wantMsg: "local records returned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirClient := &client.Client{
				RoutingServiceClient: &mockDoctorRoutingClient{
					listErr:  tt.listErr,
					recvResp: tt.recvResp,
					recvErr:  tt.recvErr,
				},
			}

			result := routingList(context.Background(), dirClient, "localhost:8888", time.Second)

			assert.Equal(t, "routing_list", result.Name)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Contains(t, result.Message, tt.wantMsg)
		})
	}
}

type mockDoctorRoutingClient struct {
	routingv1.RoutingServiceClient
	listErr  error
	recvResp *routingv1.ListResponse
	recvErr  error
}

func (m *mockDoctorRoutingClient) List(context.Context, *routingv1.ListRequest, ...grpc.CallOption) (routingv1.RoutingService_ListClient, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	return &mockDoctorListStream{
		resp: m.recvResp,
		err:  m.recvErr,
	}, nil
}

type mockDoctorListStream struct {
	grpc.ClientStream
	resp *routingv1.ListResponse
	err  error
}

func (m *mockDoctorListStream) Recv() (*routingv1.ListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}
