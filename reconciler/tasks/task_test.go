// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskResult_Fields(t *testing.T) {
	now := time.Now()
	err := errors.New("task failed")
	r := TaskResult{
		TaskName:       "test-task",
		StartTime:      now,
		Duration:       time.Second,
		ItemsProcessed: 10,
		ItemsSucceeded: 8,
		ItemsFailed:    2,
		Error:          err,
	}
	assert.Equal(t, "test-task", r.TaskName)
	assert.Equal(t, now, r.StartTime)
	assert.Equal(t, time.Second, r.Duration)
	assert.Equal(t, 10, r.ItemsProcessed)
	assert.Equal(t, 8, r.ItemsSucceeded)
	assert.Equal(t, 2, r.ItemsFailed)
	assert.Same(t, err, r.Error)
}

func TestTaskConfig_ZeroValue(t *testing.T) {
	var c TaskConfig
	assert.False(t, c.Enabled)
	assert.Equal(t, time.Duration(0), c.Interval)
}

func TestTaskConfig_WithValues(t *testing.T) {
	c := TaskConfig{
		Enabled:  true,
		Interval: 5 * time.Minute,
	}
	assert.True(t, c.Enabled)
	assert.Equal(t, 5*time.Minute, c.Interval)
}
