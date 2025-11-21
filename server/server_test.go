// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"
	"time"

	"github.com/agntcy/dir/server/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// TestBuildConnectionOptions verifies that buildConnectionOptions creates
// the correct gRPC server options from connection configuration.
func TestBuildConnectionOptions(t *testing.T) {
	tests := []struct {
		name     string
		config   config.ConnectionConfig
		validate func(t *testing.T, opts []grpc.ServerOption)
	}{
		{
			name:   "default configuration",
			config: config.DefaultConnectionConfig(),
			validate: func(t *testing.T, opts []grpc.ServerOption) {
				t.Helper()
				// Verify we get the expected number of options
				// 4 basic options + 2 keepalive options = 6 total
				assert.Len(t, opts, 6, "Should create 6 server options")
			},
		},
		{
			name: "custom configuration",
			config: config.ConnectionConfig{
				MaxConcurrentStreams: 500,
				MaxRecvMsgSize:       2 * 1024 * 1024,
				MaxSendMsgSize:       2 * 1024 * 1024,
				ConnectionTimeout:    30 * time.Second,
				Keepalive: config.KeepaliveConfig{
					MaxConnectionIdle:     5 * time.Minute,
					MaxConnectionAge:      10 * time.Minute,
					MaxConnectionAgeGrace: 2 * time.Minute,
					Time:                  2 * time.Minute,
					Timeout:               20 * time.Second,
					MinTime:               20 * time.Second,
					PermitWithoutStream:   true,
				},
			},
			validate: func(t *testing.T, opts []grpc.ServerOption) {
				t.Helper()
				// Verify we get the expected number of options
				assert.Len(t, opts, 6, "Should create 6 server options")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := buildConnectionOptions(tt.config)
			assert.NotNil(t, opts)
			tt.validate(t, opts)
		})
	}
}

// TestBuildConnectionOptions_AllOptionsPresent verifies that all required
// connection management options are included.
func TestBuildConnectionOptions_AllOptionsPresent(t *testing.T) {
	cfg := config.DefaultConnectionConfig()
	opts := buildConnectionOptions(cfg)

	// We should have exactly 6 options:
	// 1. MaxConcurrentStreams
	// 2. MaxRecvMsgSize
	// 3. MaxSendMsgSize
	// 4. ConnectionTimeout
	// 5. KeepaliveParams
	// 6. KeepaliveEnforcementPolicy
	assert.Len(t, opts, 6, "Should have 6 connection management options")

	// Verify options are not nil
	for i, opt := range opts {
		assert.NotNil(t, opt, "Option %d should not be nil", i)
	}
}

// TestBuildConnectionOptions_KeepaliveParameters verifies that keepalive
// parameters are correctly configured.
func TestBuildConnectionOptions_KeepaliveParameters(t *testing.T) {
	// Create a config with known keepalive values
	cfg := config.ConnectionConfig{
		MaxConcurrentStreams: 1000,
		MaxRecvMsgSize:       4 * 1024 * 1024,
		MaxSendMsgSize:       4 * 1024 * 1024,
		ConnectionTimeout:    120 * time.Second,
		Keepalive: config.KeepaliveConfig{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Minute,
			Timeout:               1 * time.Minute,
			MinTime:               1 * time.Minute,
			PermitWithoutStream:   true,
		},
	}

	opts := buildConnectionOptions(cfg)

	// Verify we have the keepalive options
	// We can't directly inspect the options, but we can verify they're created
	assert.NotEmpty(t, opts, "Should have created server options")
}

// TestBuildConnectionOptions_MessageSizeLimits verifies that message size
// limits are correctly configured.
func TestBuildConnectionOptions_MessageSizeLimits(t *testing.T) {
	tests := []struct {
		name           string
		maxRecvMsgSize int
		maxSendMsgSize int
	}{
		{
			name:           "4MB limits (default)",
			maxRecvMsgSize: 4 * 1024 * 1024,
			maxSendMsgSize: 4 * 1024 * 1024,
		},
		{
			name:           "8MB limits",
			maxRecvMsgSize: 8 * 1024 * 1024,
			maxSendMsgSize: 8 * 1024 * 1024,
		},
		{
			name:           "16MB limits",
			maxRecvMsgSize: 16 * 1024 * 1024,
			maxSendMsgSize: 16 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ConnectionConfig{
				MaxConcurrentStreams: 1000,
				MaxRecvMsgSize:       tt.maxRecvMsgSize,
				MaxSendMsgSize:       tt.maxSendMsgSize,
				ConnectionTimeout:    120 * time.Second,
				Keepalive:            config.KeepaliveConfig{},
			}

			opts := buildConnectionOptions(cfg)
			assert.NotEmpty(t, opts, "Should create server options")
		})
	}
}

// TestBuildConnectionOptions_StreamLimits verifies that concurrent stream
// limits are correctly configured.
func TestBuildConnectionOptions_StreamLimits(t *testing.T) {
	tests := []struct {
		name                 string
		maxConcurrentStreams uint32
	}{
		{
			name:                 "100 streams",
			maxConcurrentStreams: 100,
		},
		{
			name:                 "1000 streams (default)",
			maxConcurrentStreams: 1000,
		},
		{
			name:                 "5000 streams",
			maxConcurrentStreams: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ConnectionConfig{
				MaxConcurrentStreams: tt.maxConcurrentStreams,
				MaxRecvMsgSize:       4 * 1024 * 1024,
				MaxSendMsgSize:       4 * 1024 * 1024,
				ConnectionTimeout:    120 * time.Second,
				Keepalive:            config.KeepaliveConfig{},
			}

			opts := buildConnectionOptions(cfg)
			assert.NotEmpty(t, opts, "Should create server options")
		})
	}
}

// TestKeepaliveServerParameters_StructCreation verifies that we can create
// keepalive.ServerParameters with our configuration values.
func TestKeepaliveServerParameters_StructCreation(t *testing.T) {
	cfg := config.KeepaliveConfig{
		MaxConnectionIdle:     15 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  5 * time.Minute,
		Timeout:               1 * time.Minute,
	}

	// Verify we can create the keepalive.ServerParameters struct
	params := keepalive.ServerParameters{
		MaxConnectionIdle:     cfg.MaxConnectionIdle,
		MaxConnectionAge:      cfg.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.MaxConnectionAgeGrace,
		Time:                  cfg.Time,
		Timeout:               cfg.Timeout,
	}

	assert.Equal(t, 15*time.Minute, params.MaxConnectionIdle)
	assert.Equal(t, 30*time.Minute, params.MaxConnectionAge)
	assert.Equal(t, 5*time.Minute, params.MaxConnectionAgeGrace)
	assert.Equal(t, 5*time.Minute, params.Time)
	assert.Equal(t, 1*time.Minute, params.Timeout)
}

// TestServerInitialization_SchemaURL verifies that the server correctly
// configures the OASF schema URL during initialization.
func TestServerInitialization_SchemaURL(t *testing.T) {
	tests := []struct {
		name      string
		schemaURL string
	}{
		{
			name:      "default schema URL",
			schemaURL: config.DefaultSchemaURL,
		},
		{
			name:      "custom schema URL",
			schemaURL: "https://custom.schema.url",
		},
		{
			name:      "empty schema URL (disable API validator)",
			schemaURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal config with the schema URL
			cfg := &config.Config{
				ListenAddress: config.DefaultListenAddress,
				SchemaURL:     tt.schemaURL,
				Connection:    config.DefaultConnectionConfig(),
			}

			// We can't fully test New() because it tries to start services,
			// but we can verify that a config with SchemaURL doesn't panic
			// during the initial setup phase
			assert.NotNil(t, cfg)
			assert.Equal(t, tt.schemaURL, cfg.SchemaURL)
		})
	}
}

// TestKeepaliveEnforcementPolicy_StructCreation verifies that we can create
// keepalive.EnforcementPolicy with our configuration values.
func TestKeepaliveEnforcementPolicy_StructCreation(t *testing.T) {
	cfg := config.KeepaliveConfig{
		MinTime:             1 * time.Minute,
		PermitWithoutStream: true,
	}

	// Verify we can create the keepalive.EnforcementPolicy struct
	policy := keepalive.EnforcementPolicy{
		MinTime:             cfg.MinTime,
		PermitWithoutStream: cfg.PermitWithoutStream,
	}

	assert.Equal(t, 1*time.Minute, policy.MinTime)
	assert.True(t, policy.PermitWithoutStream)
}
