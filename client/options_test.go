// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
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

func TestSetupJWTAuth_Validation(t *testing.T) {
	t.Run("should error when JWT audience is missing", func(t *testing.T) {
		// This test validates that JWT authentication requires an audience
		opts := &options{
			config: &Config{
				ServerAddress:    testServerAddr,
				SpiffeSocketPath: testSpiffeSocket,
				AuthMode:         "jwt",
				JWTAudience:      "", // Missing audience
			},
		}

		// We need a mock client to test setupJWTAuth
		// Since we can't create a real SPIFFE client without the socket,
		// we test this through withAuth which calls setupJWTAuth
		ctx := context.Background()
		opt := withAuth(ctx)
		err := opt(opts)

		// Should fail because we can't connect to SPIFFE socket
		// OR because JWT audience is missing (depending on order of checks)
		require.Error(t, err)
		// The error could be about SPIFFE connection or missing JWT audience
		t.Logf("Error (expected): %v", err)
	})
}

func TestSetupX509Auth_Validation(t *testing.T) {
	t.Run("should attempt x509 auth setup", func(t *testing.T) {
		opts := &options{
			config: &Config{
				ServerAddress:    testServerAddr,
				SpiffeSocketPath: testSpiffeSocket,
				AuthMode:         "x509",
			},
		}

		ctx := context.Background()
		opt := withAuth(ctx)
		err := opt(opts)

		// Should fail because we can't connect to SPIFFE socket
		require.Error(t, err)
		// Error should be about SPIFFE connection
		t.Logf("Error (expected): %v", err)
	})
}

func TestWithAuth_SPIFFESocketConnection(t *testing.T) {
	t.Run("should error when SPIFFE socket does not exist", func(t *testing.T) {
		// Use a non-existent socket path
		nonExistentSocket := "/tmp/non-existent-spiffe-" + t.Name() + ".sock"

		opts := &options{
			config: &Config{
				ServerAddress:    testServerAddr,
				SpiffeSocketPath: nonExistentSocket,
				AuthMode:         "jwt",
				JWTAudience:      testJWTAudience,
			},
		}

		ctx := context.Background()
		opt := withAuth(ctx)
		err := opt(opts)

		// Should error because socket doesn't exist
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create SPIFFE client")
	})

	t.Run("should error with x509 auth and non-existent socket", func(t *testing.T) {
		nonExistentSocket := "/tmp/non-existent-spiffe-x509-" + t.Name() + ".sock"

		opts := &options{
			config: &Config{
				ServerAddress:    testServerAddr,
				SpiffeSocketPath: nonExistentSocket,
				AuthMode:         "x509",
			},
		}

		ctx := context.Background()
		opt := withAuth(ctx)
		err := opt(opts)

		// Should error because socket doesn't exist
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create SPIFFE client")
	})
}

func TestWithAuth_AllAuthModes(t *testing.T) {
	testCases := []struct {
		name          string
		authMode      string
		jwtAudience   string
		expectError   bool
		errorContains string
	}{
		{
			name:          "jwt mode without socket",
			authMode:      "jwt",
			jwtAudience:   testJWTAudience,
			expectError:   true,
			errorContains: "failed to create SPIFFE client",
		},
		{
			name:          "x509 mode without socket",
			authMode:      "x509",
			jwtAudience:   "",
			expectError:   true,
			errorContains: "failed to create SPIFFE client",
		},
		{
			name:          "invalid mode without socket",
			authMode:      "invalid",
			jwtAudience:   "",
			expectError:   true,
			errorContains: "unsupported auth mode",
		},
		{
			name:          "empty mode with socket path",
			authMode:      "",
			jwtAudience:   "",
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			socketPath := ""
			if tc.authMode != "" {
				socketPath = "/tmp/test-socket-" + tc.name + ".sock"
			}

			opts := &options{
				config: &Config{
					ServerAddress:    testServerAddr,
					SpiffeSocketPath: socketPath,
					AuthMode:         tc.authMode,
					JWTAudience:      tc.jwtAudience,
				},
			}

			ctx := context.Background()
			opt := withAuth(ctx)
			err := opt(opts)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// X509-SVID Retry Logic Tests
// ============================================================================

// mockX509Source is a mock implementation of x509SourceGetter for testing.
type mockX509Source struct {
	calls     []int
	responses []mockResponse
	callCount int
	closeFunc func() error
}

type mockResponse struct {
	svid *x509svid.SVID
	err  error
}

func newMockX509Source(responses ...mockResponse) *mockX509Source {
	return &mockX509Source{
		responses: responses,
		calls:     make([]int, 0),
	}
}

func (m *mockX509Source) GetX509SVID() (*x509svid.SVID, error) {
	m.callCount++
	m.calls = append(m.calls, m.callCount)

	if len(m.responses) == 0 {
		return nil, errors.New("no mock response configured")
	}

	// Use modulo to cycle through responses if we have fewer responses than calls
	idx := (m.callCount - 1) % len(m.responses)

	return m.responses[idx].svid, m.responses[idx].err
}

func (m *mockX509Source) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}

	return nil
}

// Test retry parameters - use smaller values for faster tests.
const (
	testMaxRetries     = 3
	testInitialBackoff = 10 * time.Millisecond
	testMaxBackoff     = 100 * time.Millisecond
)

// createValidSVID creates a mock SVID with a valid SPIFFE ID.
func createValidSVID(t *testing.T) *x509svid.SVID {
	t.Helper()
	// Create a minimal valid SPIFFE ID
	id, err := spiffeid.FromString("spiffe://example.org/test/workload")
	require.NoError(t, err)

	// Create a minimal SVID with the ID
	// Note: In a real scenario, this would have certificates, but for testing
	// we only need to validate the ID field
	return &x509svid.SVID{
		ID: id,
	}
}

// createZeroIDSVID creates a mock SVID with a zero SPIFFE ID (no URI SAN).
func createZeroIDSVID(t *testing.T) *x509svid.SVID {
	t.Helper()
	// Create an SVID with zero ID (no URI SAN) - this is the zero value
	return &x509svid.SVID{
		ID: spiffeid.ID{}, // Zero value
	}
}

func TestGetX509SVIDWithRetry_ImmediateSuccess(t *testing.T) {
	t.Run("should return valid SVID on first attempt", func(t *testing.T) {
		validSVID := createValidSVID(t)
		mockSrc := newMockX509Source(mockResponse{svid: validSVID, err: nil})

		svid, err := getX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff)

		require.NoError(t, err)
		assert.NotNil(t, svid)
		assert.Equal(t, validSVID.ID, svid.ID)
		assert.Equal(t, 1, mockSrc.callCount, "should only call GetX509SVID once")
	})
}

func TestGetX509SVIDWithRetry_RetryAfterError(t *testing.T) {
	t.Run("should retry after errors and eventually succeed", func(t *testing.T) {
		validSVID := createValidSVID(t)
		mockSrc := newMockX509Source(
			mockResponse{svid: nil, err: errors.New("temporary error")},
			mockResponse{svid: nil, err: errors.New("temporary error")},
			mockResponse{svid: validSVID, err: nil},
		)

		svid, err := getX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff)

		require.NoError(t, err)
		assert.NotNil(t, svid)
		assert.Equal(t, validSVID.ID, svid.ID)
		assert.Equal(t, 3, mockSrc.callCount, "should retry 3 times")
	})
}

func TestGetX509SVIDWithRetry_RetryAfterZeroID(t *testing.T) {
	t.Run("should retry when SVID has zero SPIFFE ID", func(t *testing.T) {
		zeroIDSVID := createZeroIDSVID(t)
		validSVID := createValidSVID(t)
		mockSrc := newMockX509Source(
			mockResponse{svid: zeroIDSVID, err: nil}, // Certificate but no URI SAN
			mockResponse{svid: zeroIDSVID, err: nil}, // Still no URI SAN
			mockResponse{svid: validSVID, err: nil},  // Finally valid
		)

		svid, err := getX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff)

		require.NoError(t, err)
		assert.NotNil(t, svid)
		assert.Equal(t, validSVID.ID, svid.ID)
		assert.Equal(t, 3, mockSrc.callCount, "should retry until valid SVID")
	})
}

func TestGetX509SVIDWithRetry_MaxRetriesExceeded(t *testing.T) {
	t.Run("should return error after max retries with errors", func(t *testing.T) {
		mockSrc := newMockX509Source(
			mockResponse{svid: nil, err: errors.New("persistent error")},
		)

		svid, err := getX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff)

		require.Error(t, err)
		assert.Nil(t, svid)
		assert.Contains(t, err.Error(), "failed to get valid X509-SVID after 3 retries")
		assert.Contains(t, err.Error(), "persistent error")
		assert.Equal(t, testMaxRetries, mockSrc.callCount, "should retry max 3 times")
	})

	t.Run("should return error after max retries with zero ID", func(t *testing.T) {
		zeroIDSVID := createZeroIDSVID(t)
		mockSrc := newMockX509Source(
			mockResponse{svid: zeroIDSVID, err: nil}, // Certificate but no URI SAN
		)

		svid, err := getX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff)

		require.Error(t, err)
		assert.Nil(t, svid)
		assert.Contains(t, err.Error(), "failed to get valid X509-SVID after 3 retries")
		assert.Contains(t, err.Error(), "certificate contains no URI SAN")
		assert.Equal(t, testMaxRetries, mockSrc.callCount, "should retry max 3 times")
	})
}

func TestGetX509SVIDWithRetry_NilSVID(t *testing.T) {
	t.Run("should retry when GetX509SVID returns nil SVID", func(t *testing.T) {
		validSVID := createValidSVID(t)
		mockSrc := newMockX509Source(
			mockResponse{svid: nil, err: nil}, // Nil SVID, no error
			mockResponse{svid: validSVID, err: nil},
		)

		svid, err := getX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff)

		require.NoError(t, err)
		assert.NotNil(t, svid)
		assert.Equal(t, validSVID.ID, svid.ID)
		assert.Equal(t, 2, mockSrc.callCount, "should retry after nil SVID")
	})
}

func TestGetX509SVIDWithRetry_ExponentialBackoff(t *testing.T) {
	t.Run("should use exponential backoff between retries", func(t *testing.T) {
		// This test verifies that backoff is applied with test parameters
		validSVID := createValidSVID(t)
		mockSrc := newMockX509Source(
			mockResponse{svid: nil, err: errors.New("error 1")},
			mockResponse{svid: nil, err: errors.New("error 2")},
			mockResponse{svid: validSVID, err: nil},
		)

		start := time.Now()
		svid, err := getX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, svid)
		// Verify that some backoff was applied (at least 10ms + 20ms = 30ms)
		// But allow some tolerance for test execution time
		assert.GreaterOrEqual(t, duration, 20*time.Millisecond, "should have applied exponential backoff")
		assert.Equal(t, 3, mockSrc.callCount, "should retry 3 times")
	})
}
