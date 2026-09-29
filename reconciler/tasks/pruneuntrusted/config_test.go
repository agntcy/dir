// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pruneuntrusted

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_GetInterval(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultInterval, (&Config{}).GetInterval())
	assert.Equal(t, 15*time.Minute, (&Config{Interval: 15 * time.Minute}).GetInterval())
}

func TestConfig_GetRecordTimeout(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultRecordTimeout, (&Config{}).GetRecordTimeout())
	assert.Equal(t, 10*time.Second, (&Config{RecordTimeout: 10 * time.Second}).GetRecordTimeout())
}

func TestConfig_GetScanSeverity(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultScanSeverity, (&Config{}).GetScanSeverity())
	assert.Equal(t, "HIGH", (&Config{ScanSeverity: "high"}).GetScanSeverity())
	assert.Equal(t, "CRITICAL", (&Config{ScanSeverity: "CRITICAL"}).GetScanSeverity())
}
