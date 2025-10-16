// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:nilnil
package config

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/client/streaming"
)

// mockClient is a mock implementation of ClientInterface for testing.
type mockClient struct{}

func (m *mockClient) PushStream(ctx context.Context, recordsCh <-chan *corev1.Record) (streaming.StreamResult[corev1.RecordRef], error) {
	return newMockStreamResult(ctx, recordsCh), nil
}

// mockStreamResult is a mock implementation of StreamResult for testing.
type mockStreamResult struct {
	resCh  chan *corev1.RecordRef
	errCh  chan error
	doneCh chan struct{}
}

func newMockStreamResult(_ context.Context, _ <-chan *corev1.Record) *mockStreamResult {
	result := &mockStreamResult{
		resCh:  make(chan *corev1.RecordRef),
		errCh:  make(chan error),
		doneCh: make(chan struct{}),
	}

	// Consume the input channel in a goroutine
	go func() {
		defer close(result.doneCh)
		defer close(result.resCh)
		defer close(result.errCh)
	}()

	return result
}

func (m *mockStreamResult) ResCh() <-chan *corev1.RecordRef {
	return m.resCh
}

func (m *mockStreamResult) ErrCh() <-chan error {
	return m.errCh
}

func (m *mockStreamResult) DoneCh() <-chan struct{} {
	return m.doneCh
}

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
				Client:       &mockClient{},
			},
			wantErr: false,
		},
		{
			name: "missing registry type",
			config: Config{
				RegistryURL: "https://registry.example.com",
				Concurrency: 10,
				Client:      &mockClient{},
			},
			wantErr: true,
			errMsg:  "registry type is required",
		},
		{
			name: "missing registry URL",
			config: Config{
				RegistryType: RegistryTypeMCP,
				Concurrency:  10,
				Client:       &mockClient{},
			},
			wantErr: true,
			errMsg:  "registry URL is required",
		},
		{
			name: "missing client",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  10,
			},
			wantErr: true,
			errMsg:  "client is required",
		},
		{
			name: "zero concurrency sets default",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  0,
				Client:       &mockClient{},
			},
			wantErr: false,
		},
		{
			name: "negative concurrency sets default",
			config: Config{
				RegistryType: RegistryTypeMCP,
				RegistryURL:  "https://registry.example.com",
				Concurrency:  -1,
				Client:       &mockClient{},
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
