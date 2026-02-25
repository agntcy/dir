// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/agntcy/dir/reconciler/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTask implements tasks.Task for testing.
type mockTask struct {
	name     string
	interval time.Duration
	enabled  bool
	runErr   error
	runCalls int
	runMu    sync.Mutex
}

func (m *mockTask) Name() string            { return m.name }
func (m *mockTask) Interval() time.Duration { return m.interval }
func (m *mockTask) IsEnabled() bool         { return m.enabled }
func (m *mockTask) Run(ctx context.Context) error {
	m.runMu.Lock()
	m.runCalls++
	m.runMu.Unlock()

	return m.runErr
}

func TestNew(t *testing.T) {
	s := New()
	require.NotNil(t, s)
	assert.NotNil(t, s.stopCh)
	assert.Empty(t, s.tasks)
}

func TestRegisterTask(t *testing.T) {
	s := New()
	task := &mockTask{name: "test", interval: time.Second, enabled: true}

	s.RegisterTask(task)

	require.Len(t, s.tasks, 1)
	assert.Same(t, task, s.tasks[0])
}

func TestRegisterTask_Multiple(t *testing.T) {
	s := New()
	t1 := &mockTask{name: "task1", interval: time.Second, enabled: true}
	t2 := &mockTask{name: "task2", interval: 2 * time.Second, enabled: false}

	s.RegisterTask(t1)
	s.RegisterTask(t2)

	require.Len(t, s.tasks, 2)
	assert.Same(t, t1, s.tasks[0])
	assert.Same(t, t2, s.tasks[1])
}

func TestIsReady(t *testing.T) {
	t.Run("no tasks", func(t *testing.T) {
		s := New()
		assert.False(t, s.IsReady(context.Background()))
	})

	t.Run("with tasks", func(t *testing.T) {
		s := New()
		s.RegisterTask(&mockTask{name: "t", interval: time.Second, enabled: true})
		assert.True(t, s.IsReady(context.Background()))
	})
}

func TestStart_StartsOnlyEnabledTasks(t *testing.T) {
	s := New()
	enabled := &mockTask{name: "enabled", interval: 10 * time.Millisecond, enabled: true}
	disabled := &mockTask{name: "disabled", interval: time.Second, enabled: false}

	s.RegisterTask(disabled)
	s.RegisterTask(enabled)

	ctx := t.Context()

	err := s.Start(ctx)
	require.NoError(t, err)

	// Give the enabled task a chance to run at least once
	time.Sleep(30 * time.Millisecond)

	enabled.runMu.Lock()
	calls := enabled.runCalls
	enabled.runMu.Unlock()
	assert.GreaterOrEqual(t, calls, 1, "enabled task should have run at least once")

	disabled.runMu.Lock()
	assert.Equal(t, 0, disabled.runCalls, "disabled task should not run")
	disabled.runMu.Unlock()

	// Stop to clean up
	s.Stop() //nolint:errcheck
}

func TestStart_ContextCancelStopsTaskLoop(t *testing.T) {
	s := New()
	task := &mockTask{name: "loop", interval: 5 * time.Millisecond, enabled: true}
	s.RegisterTask(task)

	ctx, cancel := context.WithCancel(context.Background())
	err := s.Start(ctx)
	require.NoError(t, err)

	time.Sleep(15 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)

	// Stop to release WaitGroup
	s.Stop() //nolint:errcheck

	task.runMu.Lock()
	calls := task.runCalls
	task.runMu.Unlock()
	assert.GreaterOrEqual(t, calls, 1)
}

// Ensure mockTask satisfies tasks.Task.
var _ tasks.Task = (*mockTask)(nil)
