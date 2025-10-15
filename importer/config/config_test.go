// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:nilnil
package config

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// mockStoreClient is a mock implementation of StoreServiceClient for testing.
type mockStoreClient struct{}

func (m *mockStoreClient) Push(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_PushClient, error) {
	return &mockPushClient{}, nil
}

func (m *mockStoreClient) Pull(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_PullClient, error) {
	return nil, nil
}

func (m *mockStoreClient) Lookup(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_LookupClient, error) {
	return nil, nil
}

func (m *mockStoreClient) Delete(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_DeleteClient, error) {
	return nil, nil
}

func (m *mockStoreClient) PushReferrer(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_PushReferrerClient, error) {
	return nil, nil
}

func (m *mockStoreClient) PullReferrer(ctx context.Context, opts ...grpc.CallOption) (storev1.StoreService_PullReferrerClient, error) {
	return nil, nil
}

// Mock streaming client.
type mockPushClient struct{ grpc.ClientStream }

func (m *mockPushClient) Send(r *corev1.Record) error           { return nil }
func (m *mockPushClient) Recv() (*corev1.RecordRef, error)      { return &corev1.RecordRef{}, nil }
func (m *mockPushClient) CloseSend() error                      { return nil }
func (m *mockPushClient) CloseAndRecv() (*emptypb.Empty, error) { return &emptypb.Empty{}, nil }

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  10,
				StoreClient:  &mockStoreClient{},
			},
			wantErr: false,
		},
		{
			name: "missing registry type",
			config: Config{
				RegistryURL: "https://registry.example.com",
				Concurrency: 10,
				StoreClient: &mockStoreClient{},
			},
			wantErr: true,
			errMsg:  "registry type is required",
		},
		{
			name: "missing registry URL",
			config: Config{
				RegistryType: RegistryTypeMCP,
				Concurrency:  10,
				StoreClient:  &mockStoreClient{},
			},
			wantErr: true,
			errMsg:  "registry URL is required",
		},
		{
			name: "missing store client",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  10,
			},
			wantErr: true,
			errMsg:  "store client is required",
		},
		{
			name: "zero concurrency sets default",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  0,
				StoreClient:  &mockStoreClient{},
			},
			wantErr: false,
		},
		{
			name: "negative concurrency sets default",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  -1,
				StoreClient:  &mockStoreClient{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Config.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}

			// Check that default concurrency is set when invalid
			if !tt.wantErr && tt.config.Concurrency <= 0 {
				if tt.config.Concurrency != 5 {
					t.Errorf("Config.Validate() did not set default concurrency, got %d, want 5", tt.config.Concurrency)
				}
			}
		})
	}
}
