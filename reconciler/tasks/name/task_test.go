// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package name

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Name_Interval_IsEnabled(t *testing.T) {
	task, err := NewTask(Config{Enabled: true, Interval: 2 * time.Hour}, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, "name", task.Name())
	assert.Equal(t, 2*time.Hour, task.Interval())
	assert.True(t, task.IsEnabled())
}

func TestTask_IsEnabled_False(t *testing.T) {
	task, err := NewTask(Config{Enabled: false}, nil, nil, nil)
	require.NoError(t, err)
	assert.False(t, task.IsEnabled())
}

func TestTask_Interval_Zero_UsesDefault(t *testing.T) {
	task, err := NewTask(Config{Enabled: true, Interval: 0}, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, DefaultInterval, task.Interval())
}
