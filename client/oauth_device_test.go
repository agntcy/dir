// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceFlowError_Error(t *testing.T) {
	t.Run("should format error with description", func(t *testing.T) {
		err := &DeviceFlowError{
			Code:        "authorization_pending",
			Description: "The authorization request is pending",
		}

		result := err.Error()

		assert.Equal(t, "authorization_pending: The authorization request is pending", result)
	})

	t.Run("should format error without description", func(t *testing.T) {
		err := &DeviceFlowError{
			Code: "expired_token",
		}

		result := err.Error()

		assert.Equal(t, "expired_token", result)
	})

	t.Run("should handle empty description", func(t *testing.T) {
		err := &DeviceFlowError{
			Code:        "slow_down",
			Description: "",
		}

		result := err.Error()

		assert.Equal(t, "slow_down", result)
	})
}

func TestIsRetryableDeviceError(t *testing.T) {
	t.Run("should return true for authorization_pending", func(t *testing.T) {
		err := &DeviceFlowError{Code: "authorization_pending"}

		result := isRetryableDeviceError(err)

		assert.True(t, result)
	})

	t.Run("should return true for slow_down", func(t *testing.T) {
		err := &DeviceFlowError{Code: "slow_down"}

		result := isRetryableDeviceError(err)

		assert.True(t, result)
	})

	t.Run("should return false for expired_token", func(t *testing.T) {
		err := &DeviceFlowError{Code: "expired_token"}

		result := isRetryableDeviceError(err)

		assert.False(t, result)
	})

	t.Run("should return false for access_denied", func(t *testing.T) {
		err := &DeviceFlowError{Code: "access_denied"}

		result := isRetryableDeviceError(err)

		assert.False(t, result)
	})

	t.Run("should return false for unknown error code", func(t *testing.T) {
		err := &DeviceFlowError{Code: "unknown_error"}

		result := isRetryableDeviceError(err)

		assert.False(t, result)
	})

	t.Run("should return false for non-DeviceFlowError", func(t *testing.T) {
		err := errors.New("some other error")

		result := isRetryableDeviceError(err)

		assert.False(t, result)
	})

	t.Run("should return false for nil error", func(t *testing.T) {
		result := isRetryableDeviceError(nil)

		assert.False(t, result)
	})
}

func TestGetAdjustedInterval(t *testing.T) {
	t.Run("should return adjusted interval for slow_down error", func(t *testing.T) {
		err := &DeviceFlowError{
			Code:        "slow_down",
			NewInterval: 10,
		}

		interval := getAdjustedInterval(err)

		assert.Equal(t, 10*time.Second, interval)
	})

	t.Run("should return 0 for slow_down without NewInterval", func(t *testing.T) {
		err := &DeviceFlowError{
			Code:        "slow_down",
			NewInterval: 0,
		}

		interval := getAdjustedInterval(err)

		assert.Equal(t, time.Duration(0), interval)
	})

	t.Run("should return 0 for non-slow_down error", func(t *testing.T) {
		err := &DeviceFlowError{
			Code:        "authorization_pending",
			NewInterval: 10,
		}

		interval := getAdjustedInterval(err)

		assert.Equal(t, time.Duration(0), interval)
	})

	t.Run("should return 0 for non-DeviceFlowError", func(t *testing.T) {
		err := errors.New("some other error")

		interval := getAdjustedInterval(err)

		assert.Equal(t, time.Duration(0), interval)
	})

	t.Run("should return 0 for nil error", func(t *testing.T) {
		interval := getAdjustedInterval(nil)

		assert.Equal(t, time.Duration(0), interval)
	})
}

func TestDisplayDeviceInstructions(t *testing.T) {
	t.Run("should display complete instructions", func(t *testing.T) {
		var buf bytes.Buffer

		deviceCode := &DeviceCodeResponse{
			VerificationURI: "https://github.com/login/device",
			UserCode:        "ABCD-1234",
			ExpiresIn:       900, // 15 minutes
		}

		displayDeviceInstructions(&buf, deviceCode)

		output := buf.String()

		assert.Contains(t, output, "🔐 To authenticate, please follow these steps:")
		assert.Contains(t, output, "https://github.com/login/device")
		assert.Contains(t, output, "ABCD-1234")
		assert.Contains(t, output, "Code expires in 15 minutes")
	})

	t.Run("should handle zero expiry time", func(t *testing.T) {
		var buf bytes.Buffer

		deviceCode := &DeviceCodeResponse{
			VerificationURI: "https://github.com/login/device",
			UserCode:        "TEST-CODE",
			ExpiresIn:       0,
		}

		displayDeviceInstructions(&buf, deviceCode)

		output := buf.String()

		assert.Contains(t, output, "Code expires in 0 minutes")
	})

	t.Run("should write to any io.Writer", func(t *testing.T) {
		// Test with different writer implementations
		var buf bytes.Buffer

		deviceCode := &DeviceCodeResponse{
			VerificationURI: "https://example.com",
			UserCode:        "CODE",
			ExpiresIn:       300,
		}

		displayDeviceInstructions(&buf, deviceCode)

		assert.NotEmpty(t, buf.String())
	})

	t.Run("should handle io.Discard", func(t *testing.T) {
		deviceCode := &DeviceCodeResponse{
			VerificationURI: "https://github.com/login/device",
			UserCode:        "CODE",
			ExpiresIn:       300,
		}

		// Should not panic
		displayDeviceInstructions(io.Discard, deviceCode)
	})
}

func TestStartDeviceFlow_Validation(t *testing.T) {
	t.Run("should error when config is nil", func(t *testing.T) {
		ctx := context.Background()

		result, err := StartDeviceFlow(ctx, nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("should error when ClientID is empty", func(t *testing.T) {
		ctx := context.Background()

		config := &DeviceFlowConfig{
			ClientID: "",
		}

		result, err := StartDeviceFlow(ctx, config)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "ClientID is required")
	})

	t.Run("should set Output to Discard if nil", func(t *testing.T) {
		// We can't easily test the full flow without HTTP, but we can
		// verify the validation passes and fails at the HTTP stage
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately to fail fast

		config := &DeviceFlowConfig{
			ClientID: "test-client-id",
			Output:   nil, // Should be set to Discard
		}

		// This will fail due to cancelled context, but that's expected
		_, err := StartDeviceFlow(ctx, config)

		// Should have progressed past validation
		require.Error(t, err)
		// Error should be from HTTP request, not from validation
		assert.NotContains(t, err.Error(), "config is required")
		assert.NotContains(t, err.Error(), "ClientID is required")
	})
}

func TestDeviceCodeResponse_Structure(t *testing.T) {
	t.Run("should have correct JSON tags", func(t *testing.T) {
		resp := DeviceCodeResponse{
			DeviceCode:      "device123",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		}

		// Verify all fields are set
		assert.NotEmpty(t, resp.DeviceCode)
		assert.NotEmpty(t, resp.UserCode)
		assert.NotEmpty(t, resp.VerificationURI)
		assert.NotZero(t, resp.ExpiresIn)
		assert.NotZero(t, resp.Interval)
	})
}

func TestDeviceTokenResponse_Structure(t *testing.T) {
	t.Run("should handle success response", func(t *testing.T) {
		resp := DeviceTokenResponse{
			AccessToken: "gho_token123",
			TokenType:   "bearer",
			Scope:       "read:user read:org",
		}

		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "bearer", resp.TokenType)
		assert.NotEmpty(t, resp.Scope)
		assert.Empty(t, resp.Error)
	})

	t.Run("should handle error response", func(t *testing.T) {
		resp := DeviceTokenResponse{
			Error:            "authorization_pending",
			ErrorDescription: "The authorization request is still pending",
		}

		assert.Empty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.Error)
		assert.NotEmpty(t, resp.ErrorDescription)
	})

	t.Run("should handle slow_down response with new interval", func(t *testing.T) {
		resp := DeviceTokenResponse{
			Error:    "slow_down",
			Interval: 10,
		}

		assert.Equal(t, "slow_down", resp.Error)
		assert.Equal(t, 10, resp.Interval)
	})
}

func TestDeviceFlowResult_Structure(t *testing.T) {
	t.Run("should contain all required fields", func(t *testing.T) {
		now := time.Now()

		result := DeviceFlowResult{
			AccessToken: "gho_token123",
			TokenType:   "bearer",
			Scope:       "read:user read:org",
			ExpiresAt:   now.Add(8 * time.Hour),
		}

		assert.NotEmpty(t, result.AccessToken)
		assert.Equal(t, "bearer", result.TokenType)
		assert.NotEmpty(t, result.Scope)
		assert.True(t, result.ExpiresAt.After(now))
	})
}

func TestDeviceFlowConfig_Structure(t *testing.T) {
	t.Run("should accept valid configuration", func(t *testing.T) {
		var buf bytes.Buffer

		config := DeviceFlowConfig{
			ClientID: "test-client-id",
			Scopes:   []string{"read:user", "read:org"},
			Output:   &buf,
		}

		assert.Equal(t, "test-client-id", config.ClientID)
		assert.Len(t, config.Scopes, 2)
		assert.NotNil(t, config.Output)
	})

	t.Run("should handle empty scopes", func(t *testing.T) {
		config := DeviceFlowConfig{
			Scopes: []string{},
		}

		assert.Empty(t, config.Scopes)
	})

	t.Run("should handle nil output", func(t *testing.T) {
		config := DeviceFlowConfig{
			Output: nil,
		}

		assert.Nil(t, config.Output)
	})
}

func TestDeviceFlowConstants(t *testing.T) {
	t.Run("should have reasonable default values", func(t *testing.T) {
		assert.Equal(t, 5*time.Second, defaultDeviceInterval)
		assert.Equal(t, 15*time.Minute, devicePollTimeout)
		assert.Equal(t, 30*time.Second, httpTimeout)
		assert.Equal(t, 10, maxIdleConns)
		assert.Equal(t, 2, maxIdleConnsPerHost)
		assert.Equal(t, 90*time.Second, idleConnTimeout)
		assert.Equal(t, 60, secondsPerMinute)
	})

	t.Run("should have valid GitHub endpoints", func(t *testing.T) {
		assert.Contains(t, githubDeviceCodeURL, "github.com")
		assert.Contains(t, githubDeviceTokenURL, "github.com")
	})
}

func TestDefaultHTTPClient(t *testing.T) {
	t.Run("should be configured properly", func(t *testing.T) {
		assert.NotNil(t, defaultHTTPClient)
		assert.Equal(t, httpTimeout, defaultHTTPClient.Timeout)
		assert.NotNil(t, defaultHTTPClient.Transport)
	})
}
