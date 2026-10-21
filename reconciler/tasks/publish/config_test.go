// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_GetInterval(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultInterval, (&Config{}).GetInterval())
	assert.Equal(t, 6*time.Hour, (&Config{Interval: 6 * time.Hour}).GetInterval())
}

func TestConfig_GetLimit(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultLimit, (&Config{}).GetLimit())
	assert.Equal(t, DefaultLimit, (&Config{Limit: -1}).GetLimit())
	assert.Equal(t, 25, (&Config{Limit: 25}).GetLimit())
}

func TestConfig_GetDomains(t *testing.T) {
	t.Parallel()

	assert.Empty(t, (&Config{}).GetDomains())
	assert.Empty(t, (&Config{Criteria: Criteria{Domains: []string{"  "}}}).GetDomains())
	assert.Equal(t, []string{"research"}, (&Config{Criteria: Criteria{Domains: []string{" research "}}}).GetDomains())
	assert.Equal(t, []string{"research", "analytics"}, (&Config{Criteria: Criteria{Domains: []string{"research", " analytics ", "research", ""}}}).GetDomains())
}

func TestConfig_GetSkills(t *testing.T) {
	t.Parallel()

	assert.Empty(t, (&Config{}).GetSkills())
	assert.Empty(t, (&Config{Criteria: Criteria{Skills: []string{"  "}}}).GetSkills())
	assert.Equal(t, []string{"threat_detection"}, (&Config{Criteria: Criteria{Skills: []string{" threat_detection "}}}).GetSkills())
	assert.Equal(t, []string{"threat_detection", "nlp"}, (&Config{Criteria: Criteria{Skills: []string{"threat_detection", " nlp ", "threat_detection", ""}}}).GetSkills())
}

func TestConfig_TrustedLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "unset", (&Config{}).trustedLabel())

	untrusted := false
	assert.Equal(t, "false", (&Config{Criteria: Criteria{Trusted: &untrusted}}).trustedLabel())

	trusted := true
	assert.Equal(t, "true", (&Config{Criteria: Criteria{Trusted: &trusted}}).trustedLabel())
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	assert.NoError(t, (&Config{}).Validate())
	assert.NoError(t, (&Config{Interval: 0, Limit: 0}).Validate())
	assert.NoError(t, (&Config{Criteria: Criteria{Domains: []string{"research"}}}).Validate())
	assert.NoError(t, (&Config{Interval: time.Hour, Limit: 25}).Validate())

	require.EqualError(t, (&Config{Interval: -time.Second}).Validate(), "invalid interval -1s, must be >= 0")
	require.EqualError(t, (&Config{Limit: -1}).Validate(), "invalid limit -1, must be >= 0")
}
