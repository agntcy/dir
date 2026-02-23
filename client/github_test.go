// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants.
const (
	testGitHubToken = "gho_testtoken123"
)

func TestNewGitHubCredentials(t *testing.T) {
	t.Run("should create credentials with token", func(t *testing.T) {
		token := "gho_testtoken123456789"
		creds := newGitHubCredentials(token)

		require.NotNil(t, creds)

		// Verify it's the correct type
		githubCreds, ok := creds.(*githubPerRPCCredentials)
		require.True(t, ok, "credentials should be of type *githubPerRPCCredentials")
		assert.Equal(t, token, githubCreds.token)
	})

	t.Run("should create credentials with empty token", func(t *testing.T) {
		// Should be allowed - validation happens elsewhere
		creds := newGitHubCredentials("")

		require.NotNil(t, creds)

		githubCreds, ok := creds.(*githubPerRPCCredentials)
		require.True(t, ok)
		assert.Empty(t, githubCreds.token)
	})

	t.Run("should create credentials with PAT token", func(t *testing.T) {
		token := "ghp_pattoken123456789"
		creds := newGitHubCredentials(token)

		require.NotNil(t, creds)

		githubCreds, ok := creds.(*githubPerRPCCredentials)
		require.True(t, ok)
		assert.Equal(t, token, githubCreds.token)
	})
}

func TestGitHubPerRPCCredentials_GetRequestMetadata(t *testing.T) {
	t.Run("should add Bearer token to authorization header", func(t *testing.T) {
		token := testGitHubToken
		creds := &githubPerRPCCredentials{token: token}

		ctx := context.Background()
		metadata, err := creds.GetRequestMetadata(ctx)

		require.NoError(t, err)
		require.NotNil(t, metadata)
		assert.Contains(t, metadata, "authorization")
		assert.Equal(t, "Bearer "+token, metadata["authorization"])
	})

	t.Run("should work with empty token", func(t *testing.T) {
		creds := &githubPerRPCCredentials{token: ""}

		ctx := context.Background()
		metadata, err := creds.GetRequestMetadata(ctx)

		require.NoError(t, err)
		require.NotNil(t, metadata)
		assert.Equal(t, "Bearer ", metadata["authorization"])
	})

	t.Run("should work with cancelled context", func(t *testing.T) {
		token := testGitHubToken
		creds := &githubPerRPCCredentials{token: token}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Should still work because GetRequestMetadata doesn't use the context
		metadata, err := creds.GetRequestMetadata(ctx)

		require.NoError(t, err)
		require.NotNil(t, metadata)
		assert.Equal(t, "Bearer "+token, metadata["authorization"])
	})

	t.Run("should work regardless of URI parameter", func(t *testing.T) {
		token := testGitHubToken
		creds := &githubPerRPCCredentials{token: token}

		ctx := context.Background()

		// Test with no URI
		metadata1, err := creds.GetRequestMetadata(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Bearer "+token, metadata1["authorization"])

		// Test with one URI
		metadata2, err := creds.GetRequestMetadata(ctx, "grpc://example.com/service/method")
		require.NoError(t, err)
		assert.Equal(t, "Bearer "+token, metadata2["authorization"])

		// Test with multiple URIs
		metadata3, err := creds.GetRequestMetadata(ctx, "uri1", "uri2", "uri3")
		require.NoError(t, err)
		assert.Equal(t, "Bearer "+token, metadata3["authorization"])

		// All should return the same metadata
		assert.Equal(t, metadata1, metadata2)
		assert.Equal(t, metadata1, metadata3)
	})

	t.Run("should create new map for each call", func(t *testing.T) {
		token := testGitHubToken
		creds := &githubPerRPCCredentials{token: token}

		ctx := context.Background()
		metadata1, err1 := creds.GetRequestMetadata(ctx)
		metadata2, err2 := creds.GetRequestMetadata(ctx)

		require.NoError(t, err1)
		require.NoError(t, err2)

		// Should be equal but different map instances
		assert.Equal(t, metadata1, metadata2)

		// Modify one shouldn't affect the other
		metadata1["test"] = "value"

		assert.NotContains(t, metadata2, "test")
	})

	t.Run("should handle long tokens", func(t *testing.T) {
		// Create a very long token (simulating a JWT or similar)
		longToken := "gho_" + string(make([]byte, 10000))
		creds := &githubPerRPCCredentials{token: longToken}

		ctx := context.Background()
		metadata, err := creds.GetRequestMetadata(ctx)

		require.NoError(t, err)
		assert.Equal(t, "Bearer "+longToken, metadata["authorization"])
	})
}

//nolint:gosec // G117: intentional test token
func TestGitHubPerRPCCredentials_RequireTransportSecurity(t *testing.T) {
	t.Run("should return false for insecure transport", func(t *testing.T) {
		creds := &githubPerRPCCredentials{token: "gho_test"}

		secure := creds.RequireTransportSecurity()

		assert.False(t, secure, "GitHub credentials should work over insecure connections (Envoy handles TLS)")
	})

	t.Run("should return false regardless of token", func(t *testing.T) {
		testCases := []struct {
			name  string
			token string
		}{
			{
				name:  "with OAuth token",
				token: "gho_oauth123",
			},
			{
				name:  "with PAT token",
				token: "ghp_pat123",
			},
			{
				name:  "with empty token",
				token: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				creds := &githubPerRPCCredentials{token: tc.token}
				assert.False(t, creds.RequireTransportSecurity())
			})
		}
	})
}

func TestGitHubPerRPCCredentials_Integration(t *testing.T) {
	t.Run("should work as PerRPCCredentials interface", func(t *testing.T) {
		token := "gho_integration_test"

		// Create via the public constructor
		var creds any = newGitHubCredentials(token)

		// Verify it implements the interface
		_, ok := creds.(interface {
			GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error)
			RequireTransportSecurity() bool
		})

		assert.True(t, ok, "should implement PerRPCCredentials interface methods")
	})

	t.Run("should be usable in gRPC dial options", func(t *testing.T) {
		token := "gho_grpc_test" //nolint:gosec // G101: This is a test token, not a real credential

		creds := newGitHubCredentials(token)

		// Verify we can call the interface methods
		ctx := context.Background()
		metadata, err := creds.GetRequestMetadata(ctx, "grpc://test")

		require.NoError(t, err)
		assert.Equal(t, "Bearer "+token, metadata["authorization"])
		assert.False(t, creds.RequireTransportSecurity())
	})
}

//nolint:gosec // G117: intentional test token
func TestGitHubPerRPCCredentials_TokenFormats(t *testing.T) {
	t.Run("should handle various GitHub token formats", func(t *testing.T) {
		testCases := []struct {
			name     string
			token    string
			expected string
		}{
			{
				name:     "OAuth token (gho_)",
				token:    "gho_16C7e42F292c6912E7710c838347Ae178B4a",
				expected: "Bearer gho_16C7e42F292c6912E7710c838347Ae178B4a",
			},
			{
				name:     "PAT classic (ghp_)",
				token:    "ghp_1234567890abcdefghijklmnopqrstuvwxyz",
				expected: "Bearer ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			},
			{
				name:     "App installation token (ghs_)",
				token:    "ghs_16C7e42F292c6912E7710c838347Ae178B4a",
				expected: "Bearer ghs_16C7e42F292c6912E7710c838347Ae178B4a",
			},
			{
				name:     "User-to-server token (ghu_)",
				token:    "ghu_16C7e42F292c6912E7710c838347Ae178B4a",
				expected: "Bearer ghu_16C7e42F292c6912E7710c838347Ae178B4a",
			},
			{
				name:     "Server-to-server token (ghs_)",
				token:    "ghs_16C7e42F292c6912E7710c838347Ae178B4a",
				expected: "Bearer ghs_16C7e42F292c6912E7710c838347Ae178B4a",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				creds := &githubPerRPCCredentials{token: tc.token}

				ctx := context.Background()
				metadata, err := creds.GetRequestMetadata(ctx)

				require.NoError(t, err)
				assert.Equal(t, tc.expected, metadata["authorization"])
			})
		}
	})
}
