// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants.
const (
	testServerAddr      = "localhost:9999"
	testSpiffeSocket    = "/tmp/test-spiffe.sock"
	testJWTAudience     = "test-audience"
	testInvalidAuthMode = "invalid-auth"
)

func TestWithConfig(t *testing.T) {
	t.Run("should set config", func(t *testing.T) {
		cfg := &Config{
			ServerAddress: testServerAddr,
		}

		opts := &options{}
		opt := WithConfig(cfg)
		err := opt(opts)

		require.NoError(t, err)
		assert.Equal(t, cfg, opts.config)
		assert.Equal(t, testServerAddr, opts.config.ServerAddress)
	})

	t.Run("should allow nil config", func(t *testing.T) {
		opts := &options{}
		opt := WithConfig(nil)
		err := opt(opts)

		require.NoError(t, err)
		assert.Nil(t, opts.config)
	})
}

func TestWithEnvConfig(t *testing.T) {
	t.Run("should load default config when no env vars", func(t *testing.T) {
		// Clear any existing env vars by unsetting them
		// Note: We use os.Unsetenv here (not t.Setenv) because t.Setenv("VAR", "")
		// sets to empty string, not unset. We need truly unset vars to test defaults.
		oldAddr := os.Getenv("DIRECTORY_CLIENT_SERVER_ADDRESS")
		oldSocket := os.Getenv("DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH")
		oldAuth := os.Getenv("DIRECTORY_CLIENT_AUTH_MODE")
		oldAud := os.Getenv("DIRECTORY_CLIENT_JWT_AUDIENCE")

		os.Unsetenv("DIRECTORY_CLIENT_SERVER_ADDRESS")
		os.Unsetenv("DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH")
		os.Unsetenv("DIRECTORY_CLIENT_AUTH_MODE")
		os.Unsetenv("DIRECTORY_CLIENT_JWT_AUDIENCE")

		defer func() {
			// Restore original values - must use os.Setenv (not t.Setenv) to restore after os.Unsetenv
			//nolint:usetesting // Can't use t.Setenv in defer for restoration after os.Unsetenv
			if oldAddr != "" {
				os.Setenv("DIRECTORY_CLIENT_SERVER_ADDRESS", oldAddr)
			}
			//nolint:usetesting // Can't use t.Setenv in defer for restoration after os.Unsetenv
			if oldSocket != "" {
				os.Setenv("DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH", oldSocket)
			}
			//nolint:usetesting // Can't use t.Setenv in defer for restoration after os.Unsetenv
			if oldAuth != "" {
				os.Setenv("DIRECTORY_CLIENT_AUTH_MODE", oldAuth)
			}
			//nolint:usetesting // Can't use t.Setenv in defer for restoration after os.Unsetenv
			if oldAud != "" {
				os.Setenv("DIRECTORY_CLIENT_JWT_AUDIENCE", oldAud)
			}
		}()

		opts := &options{}
		opt := WithEnvConfig()
		err := opt(opts)

		require.NoError(t, err)
		require.NotNil(t, opts.config)
		assert.Equal(t, DefaultServerAddress, opts.config.ServerAddress)
		assert.Empty(t, opts.config.SpiffeSocketPath)
		assert.Empty(t, opts.config.AuthMode)
		assert.Empty(t, opts.config.JWTAudience)
	})

	t.Run("should load config from environment variables", func(t *testing.T) {
		// Set env vars - t.Setenv automatically restores after test
		t.Setenv("DIRECTORY_CLIENT_SERVER_ADDRESS", testServerAddr)
		t.Setenv("DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH", testSpiffeSocket)
		t.Setenv("DIRECTORY_CLIENT_AUTH_MODE", "jwt")
		t.Setenv("DIRECTORY_CLIENT_JWT_AUDIENCE", testJWTAudience)

		opts := &options{}
		opt := WithEnvConfig()
		err := opt(opts)

		require.NoError(t, err)
		require.NotNil(t, opts.config)
		assert.Equal(t, testServerAddr, opts.config.ServerAddress)
		assert.Equal(t, testSpiffeSocket, opts.config.SpiffeSocketPath)
		assert.Equal(t, "jwt", opts.config.AuthMode)
		assert.Equal(t, testJWTAudience, opts.config.JWTAudience)
	})
}

func TestWithAuth_ConfigValidation(t *testing.T) {
	t.Run("should error when config is nil", func(t *testing.T) {
		opts := &options{
			config: nil,
		}

		ctx := context.Background()
		opt := withAuth(ctx)
		err := opt(opts)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("should use insecure credentials when no SPIFFE socket", func(t *testing.T) {
		opts := &options{
			config: &Config{
				ServerAddress:    testServerAddr,
				SpiffeSocketPath: "", // No SPIFFE
				AuthMode:         "",
			},
		}

		ctx := context.Background()
		opt := withAuth(ctx)
		err := opt(opts)

		require.NoError(t, err)
		assert.NotEmpty(t, opts.authOpts)
		assert.Nil(t, opts.authClient)
	})

	t.Run("should use insecure credentials when no auth mode", func(t *testing.T) {
		opts := &options{
			config: &Config{
				ServerAddress:    testServerAddr,
				SpiffeSocketPath: testSpiffeSocket,
				AuthMode:         "", // No auth mode
			},
		}

		ctx := context.Background()
		opt := withAuth(ctx)
		err := opt(opts)

		require.NoError(t, err)
		assert.NotEmpty(t, opts.authOpts)
		assert.Nil(t, opts.authClient)
	})
}

func TestWithAuth_InvalidAuthMode(t *testing.T) {
	t.Run("should error on unsupported auth mode", func(t *testing.T) {
		// Skip this test if we can't connect to SPIFFE socket
		// (SPIFFE connection will fail before we can test invalid auth mode)
		if _, err := os.Stat(testSpiffeSocket); os.IsNotExist(err) {
			t.Skip("SPIFFE socket not available for testing")
		}

		opts := &options{
			config: &Config{
				ServerAddress:    testServerAddr,
				SpiffeSocketPath: testSpiffeSocket,
				AuthMode:         testInvalidAuthMode,
			},
		}

		ctx := context.Background()
		opt := withAuth(ctx)
		err := opt(opts)

		// Will error either from SPIFFE connection or invalid auth mode
		require.Error(t, err)
	})
}

func TestOptions_Chaining(t *testing.T) {
	t.Run("should apply multiple options in order", func(t *testing.T) {
		cfg1 := &Config{ServerAddress: "server1:8888"}
		cfg2 := &Config{ServerAddress: "server2:9999"}

		opts := &options{}

		// Apply first config
		opt1 := WithConfig(cfg1)
		err := opt1(opts)
		require.NoError(t, err)
		assert.Equal(t, "server1:8888", opts.config.ServerAddress)

		// Apply second config (should override)
		opt2 := WithConfig(cfg2)
		err = opt2(opts)
		require.NoError(t, err)
		assert.Equal(t, "server2:9999", opts.config.ServerAddress)
	})
}

func TestOptions_DefaultValues(t *testing.T) {
	t.Run("should use default server address", func(t *testing.T) {
		assert.Equal(t, "0.0.0.0:8888", DefaultServerAddress)
		assert.Equal(t, DefaultServerAddress, DefaultConfig.ServerAddress)
	})

	t.Run("should have correct env prefix", func(t *testing.T) {
		assert.Equal(t, "DIRECTORY_CLIENT", DefaultEnvPrefix)
	})
}

func TestOptions_ContextUsage(t *testing.T) {
	t.Run("should accept cancelled context", func(t *testing.T) {
		// Create already-cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		opts := &options{
			config: &Config{
				ServerAddress: testServerAddr,
				// No SPIFFE - should use insecure
			},
		}

		opt := withAuth(ctx)
		err := opt(opts)

		// Should succeed because no actual I/O happens with insecure mode
		require.NoError(t, err)
	})
}

func TestOptions_ResourceFields(t *testing.T) {
	t.Run("should initialize with nil resources", func(t *testing.T) {
		opts := &options{}

		assert.Nil(t, opts.config)
		assert.Nil(t, opts.authClient)
		assert.Nil(t, opts.bundleSrc)
		assert.Nil(t, opts.x509Src)
		assert.Nil(t, opts.jwtSource)
		assert.Empty(t, opts.authOpts)
	})

	t.Run("should store config correctly", func(t *testing.T) {
		cfg := &Config{
			ServerAddress:    testServerAddr,
			SpiffeSocketPath: testSpiffeSocket,
			AuthMode:         "jwt",
			JWTAudience:      testJWTAudience,
		}

		opts := &options{}
		opt := WithConfig(cfg)
		err := opt(opts)

		require.NoError(t, err)
		assert.NotNil(t, opts.config)
		assert.Equal(t, testServerAddr, opts.config.ServerAddress)
		assert.Equal(t, testSpiffeSocket, opts.config.SpiffeSocketPath)
		assert.Equal(t, "jwt", opts.config.AuthMode)
		assert.Equal(t, testJWTAudience, opts.config.JWTAudience)
	})
}
