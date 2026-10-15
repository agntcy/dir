// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, []string{DefaultDomain}, (&Config{}).GetDomains())
	assert.Equal(t, []string{DefaultDomain}, (&Config{Criteria: Criteria{Domains: []string{"  "}}}).GetDomains())
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

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	assert.NoError(t, (&Config{}).Validate())
	assert.NoError(t, (&Config{Criteria: Criteria{Domains: []string{"research"}}}).Validate())
}
