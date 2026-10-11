// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prune

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestConfig_GetLimit(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultLimit, (&Config{}).GetLimit())
	assert.Equal(t, DefaultLimit, (&Config{Limit: -1}).GetLimit())
	assert.Equal(t, 25, (&Config{Limit: 25}).GetLimit())
}

func TestConfig_GetOlderThan(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultOlderThan, (&Config{}).GetOlderThan())
	assert.Equal(t, DefaultOlderThan, (&Config{Criteria: Criteria{OlderThan: -time.Hour}}).GetOlderThan())
	assert.Equal(t, 48*time.Hour, (&Config{Criteria: Criteria{OlderThan: 48 * time.Hour}}).GetOlderThan())
}

func TestConfig_GetMinSeverity(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultMinSeverity, (&Config{}).GetMinSeverity())
	assert.Equal(t, "HIGH", (&Config{Criteria: Criteria{MinSeverity: "high"}}).GetMinSeverity())
	assert.Equal(t, "CRITICAL", (&Config{Criteria: Criteria{MinSeverity: "CRITICAL"}}).GetMinSeverity())
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	assert.NoError(t, (&Config{}).Validate())
	assert.NoError(t, (&Config{Criteria: Criteria{MinSeverity: "high"}}).Validate())
	assert.NoError(t, (&Config{Criteria: Criteria{MinSeverity: "CRITICAL"}}).Validate())

	err := (&Config{Criteria: Criteria{MinSeverity: "MEDUIM"}}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid min_severity "MEDUIM"`)
	assert.Contains(t, err.Error(), "NONE, INFO, LOW, MEDIUM, HIGH, CRITICAL")

	err = (&Config{Criteria: Criteria{MinSeverity: "UNSPECIFIED"}}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid min_severity "UNSPECIFIED"`)
}

func TestConfig_Validate_SignatureStatuses(t *testing.T) {
	t.Parallel()

	assert.NoError(t, (&Config{Criteria: Criteria{SignatureStatuses: []string{}}}).Validate())
	assert.NoError(t, (&Config{Criteria: Criteria{SignatureStatuses: []string{"verified", "failed", "unknown"}}}).Validate())

	err := (&Config{Criteria: Criteria{SignatureStatuses: []string{"failed", "unsigned"}}}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid signature_statuses value "unsigned"`)
	assert.Contains(t, err.Error(), "verified, failed, unknown")
}

func TestConfig_SignatureStatusHelpers(t *testing.T) {
	t.Parallel()

	empty := &Config{}
	assert.False(t, empty.UsesSignatureStatuses())
	assert.False(t, empty.SelectsUnknownSignatures())

	failed := &Config{Criteria: Criteria{SignatureStatuses: []string{"failed"}}}
	assert.True(t, failed.UsesSignatureStatuses())
	assert.False(t, failed.SelectsUnknownSignatures())

	unknown := &Config{Criteria: Criteria{SignatureStatuses: []string{"failed", "unknown"}}}
	assert.True(t, unknown.SelectsUnknownSignatures())
}
