// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/importer/config"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestNewImporter(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: config.Config{
				RegistryType: config.RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				StoreClient:  &mockStoreClient{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer, err := NewImporter(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewImporter() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if importer == nil {
				t.Error("NewImporter() returned nil importer")
			}
		})
	}
}

// mockStoreClient is a mock implementation of StoreServiceClient for testing.
type mockStoreClient struct{}

func (m *mockStoreClient) Push(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_PushClient, error) {
	return &mockPushClient{}, nil
}

func (m *mockStoreClient) Pull(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_PullClient, error) {
	return &mockPullClient{}, nil
}

func (m *mockStoreClient) Lookup(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_LookupClient, error) {
	return &mockLookupClient{}, nil
}

func (m *mockStoreClient) Delete(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_DeleteClient, error) {
	return &mockDeleteClient{}, nil
}

func (m *mockStoreClient) PushReferrer(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_PushReferrerClient, error) {
	return &mockPushReferrerClient{}, nil
}

func (m *mockStoreClient) PullReferrer(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_PullReferrerClient, error) {
	return &mockPullReferrerClient{}, nil
}

// Mock streaming clients.
type mockPushClient struct{ grpc.ClientStream }

func (m *mockPushClient) Send(r *corev1.Record) error { return nil }
func (m *mockPushClient) Recv() (*corev1.RecordRef, error) {
	return &corev1.RecordRef{Cid: "mock-cid"}, nil
}
func (m *mockPushClient) CloseSend() error { return nil }

type mockPullClient struct{ grpc.ClientStream }

func (m *mockPullClient) Send(r *corev1.RecordRef) error { return nil }
func (m *mockPullClient) Recv() (*corev1.Record, error)  { return &corev1.Record{}, nil }
func (m *mockPullClient) CloseSend() error               { return nil }

type mockLookupClient struct{ grpc.ClientStream }

func (m *mockLookupClient) Send(r *corev1.RecordRef) error    { return nil }
func (m *mockLookupClient) Recv() (*corev1.RecordMeta, error) { return &corev1.RecordMeta{}, nil }
func (m *mockLookupClient) CloseSend() error                  { return nil }

type mockDeleteClient struct{ grpc.ClientStream }

func (m *mockDeleteClient) Send(r *corev1.RecordRef) error { return nil }
func (m *mockDeleteClient) CloseAndRecv() (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

type mockPushReferrerClient struct{ grpc.ClientStream }

func (m *mockPushReferrerClient) Send(r *storev1.PushReferrerRequest) error { return nil }
func (m *mockPushReferrerClient) Recv() (*storev1.PushReferrerResponse, error) {
	return &storev1.PushReferrerResponse{}, nil
}
func (m *mockPushReferrerClient) CloseSend() error { return nil }

type mockPullReferrerClient struct{ grpc.ClientStream }

func (m *mockPullReferrerClient) Send(r *storev1.PullReferrerRequest) error { return nil }
func (m *mockPullReferrerClient) Recv() (*storev1.PullReferrerResponse, error) {
	return &storev1.PullReferrerResponse{}, nil
}
func (m *mockPullReferrerClient) CloseSend() error { return nil }
