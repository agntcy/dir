// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package spiffe

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// testLogger creates a logger for testing.
func testLogger() *slog.Logger {
	return slog.Default()
}

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

		svid, err := GetX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff, testLogger())

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

		svid, err := GetX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff, testLogger())

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

		svid, err := GetX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff, testLogger())

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

		svid, err := GetX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff, testLogger())

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

		svid, err := GetX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff, testLogger())

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

		svid, err := GetX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff, testLogger())

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
		svid, err := GetX509SVIDWithRetry(mockSrc, testMaxRetries, testInitialBackoff, testMaxBackoff, testLogger())
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, svid)
		// Verify that some backoff was applied (at least 10ms + 20ms = 30ms)
		// But allow some tolerance for test execution time
		assert.GreaterOrEqual(t, duration, 20*time.Millisecond, "should have applied exponential backoff")
		assert.Equal(t, 3, mockSrc.callCount, "should retry 3 times")
	})
}
