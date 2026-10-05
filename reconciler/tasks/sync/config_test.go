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

func TestConfig_GetDomain(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultDomain, (&Config{}).GetDomain())
	assert.Equal(t, DefaultDomain, (&Config{Criteria: Criteria{Domain: "  "}}).GetDomain())
	assert.Equal(t, "research", (&Config{Criteria: Criteria{Domain: " research "}}).GetDomain())
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	assert.NoError(t, (&Config{}).Validate())
	assert.NoError(t, (&Config{Criteria: Criteria{Domain: "research"}}).Validate())
}
