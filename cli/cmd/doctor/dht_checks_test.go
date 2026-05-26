// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDHTLowHangingBranches(t *testing.T) {
	result := dhtBootstrap(context.Background(), validateBootstrapPeers(nil), time.Millisecond)
	assert.Equal(t, "dht_bootstrap_reachability", result.Name)
	assert.Equal(t, statusSkip, result.Status)

	validation := validateBootstrapPeers([]string{"/dns4/bootstrap.example.com/tcp/5555"})
	result = dhtBootstrap(context.Background(), validation, time.Millisecond)
	assert.Equal(t, statusSkip, result.Status)
	assert.Contains(t, result.Message, "multiaddr validation failed")

	result = failedDHTResult("failed", assert.AnError, time.Millisecond, nil)
	assert.Equal(t, statusFail, result.Status)
	assert.Equal(t, assert.AnError.Error(), result.Details["error"])

	result = failedDHTResult("failed with details", assert.AnError, time.Millisecond, map[string]string{"detail": "value"})
	assert.Equal(t, "value", result.Details["detail"])
	assert.NotContains(t, result.Details, "error")
}

func TestAddDHTCloseDetails(t *testing.T) {
	details := map[string]string{}

	hadError := addDHTCloseDetails(details, fakeCloser{}, fakeCloser{err: assert.AnError})

	assert.True(t, hadError)
	assert.Equal(t, assert.AnError.Error(), details["host_close_error"])
	assert.NotContains(t, details, "dht_close_error")

	details = map[string]string{}
	hadError = addDHTCloseDetails(details, fakeCloser{err: assert.AnError}, fakeCloser{err: assert.AnError})

	assert.True(t, hadError)
	assert.Equal(t, assert.AnError.Error(), details["dht_close_error"])
	assert.Equal(t, assert.AnError.Error(), details["host_close_error"])
}

type fakeCloser struct {
	err error
}

func (f fakeCloser) Close() error {
	return f.err
}
