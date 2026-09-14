// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/agntcy/dir/reconciler/config"
	"github.com/agntcy/dir/reconciler/tasks"
	"github.com/agntcy/dir/reconciler/tasks/identity"
	ansconfig "github.com/agntcy/dir/server/identity/ans/config"
	servertypes "github.com/agntcy/dir/server/types"
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
func (m *mockTask) Run(_ context.Context) error {
	m.runMu.Lock()
	m.runCalls++
	m.runMu.Unlock()

	return m.runErr
}

func newTestService() *Service {
	return &Service{
		tasks:  []tasks.Task{},
		stopCh: make(chan struct{}),
	}
}

func TestNewService(t *testing.T) {
	s := newTestService()
	require.NotNil(t, s)
	assert.NotNil(t, s.stopCh)
	assert.Empty(t, s.tasks)
}

func TestAddTask(t *testing.T) {
	s := newTestService()
	task := &mockTask{name: "test", interval: time.Second, enabled: true}

	s.addTask(task)

	require.Len(t, s.tasks, 1)
	assert.Same(t, task, s.tasks[0])
}

func TestAddTask_Multiple(t *testing.T) {
	s := newTestService()
	t1 := &mockTask{name: "task1", interval: time.Second, enabled: true}
	t2 := &mockTask{name: "task2", interval: 2 * time.Second, enabled: false}

	s.addTask(t1)
	s.addTask(t2)

	require.Len(t, s.tasks, 2)
	assert.Same(t, t1, s.tasks[0])
	assert.Same(t, t2, s.tasks[1])
}

func TestIsReady(t *testing.T) {
	t.Run("no tasks", func(t *testing.T) {
		s := newTestService()
		assert.False(t, s.IsReady(context.Background()))
	})

	t.Run("with tasks", func(t *testing.T) {
		s := newTestService()
		s.addTask(&mockTask{name: "t", interval: time.Second, enabled: true})
		assert.True(t, s.IsReady(context.Background()))
	})
}

func TestStart_StartsOnlyEnabledTasks(t *testing.T) {
	s := newTestService()
	enabled := &mockTask{name: "enabled", interval: 10 * time.Millisecond, enabled: true}
	disabled := &mockTask{name: "disabled", interval: time.Second, enabled: false}

	s.addTask(disabled)
	s.addTask(enabled)

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
	s := newTestService()
	task := &mockTask{name: "loop", interval: 5 * time.Millisecond, enabled: true}
	s.addTask(task)

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

// plainStore is a store without referrer support.
type plainStore struct {
	servertypes.StoreAPI
}

// referrerStore is a store with referrer support; the task constructor only
// keeps a reference to it.
type referrerStore struct {
	servertypes.FullStore
}

func TestNewIdentityTask(t *testing.T) {
	tests := []struct {
		name     string
		cfg      identity.Config
		store    servertypes.StoreAPI
		wantTask bool
		wantErr  string
	}{
		{
			name:  "a store without referrers skips the task",
			cfg:   identity.Config{Enabled: true},
			store: plainStore{},
		},
		{
			name:    "a SPIFFE bundle that cannot be loaded fails registration",
			cfg:     identity.Config{Enabled: true, SpiffeTrustDomains: map[string]string{"acme.com": "/nonexistent/bundle.pem"}},
			store:   referrerStore{},
			wantErr: "SPIFFE trust bundles",
		},
		{
			name:     "a referrer store yields the identity task",
			cfg:      identity.Config{Enabled: true},
			store:    referrerStore{},
			wantTask: true,
		},
		{
			name:    "an ans block that fails validation stops registration",
			cfg:     identity.Config{Enabled: true, Ans: ansconfig.Config{Enabled: true}},
			store:   referrerStore{},
			wantErr: "trusted_log_hosts",
		},
		{
			name:     "an enabled ans block yields the identity task",
			cfg:      identity.Config{Enabled: true, Ans: validAnsConfig()},
			store:    referrerStore{},
			wantTask: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, ok, err := newIdentityTask(tt.cfg, nil, tt.store)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTask, ok)

			if tt.wantTask {
				assert.Equal(t, "identity", task.Name())
			} else {
				assert.Nil(t, task)
			}
		})
	}
}

// validAnsConfig is an ans block the resolver accepts without a network call.
func validAnsConfig() ansconfig.Config {
	return ansconfig.Config{
		Enabled:               true,
		TrustedLogHosts:       []string{"log.example.com"},
		AllowUnpinnedRootKeys: true,
	}
}

// New with only the identity task enabled: the ans block decides between a
// registered task and a startup error.
func TestNew_IdentityTask(t *testing.T) {
	tests := []struct {
		name      string
		ans       ansconfig.Config
		wantTasks []string
		wantErr   string
	}{
		{name: "ans off registers the identity task", wantTasks: []string{"identity"}},
		{name: "a valid ans block registers the identity task", ans: validAnsConfig(), wantTasks: []string{"identity"}},
		{name: "an invalid ans block fails New", ans: ansconfig.Config{Enabled: true}, wantErr: "trusted_log_hosts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Identity: identity.Config{Enabled: true, Ans: tt.ans}}

			svc, err := New(cfg, nil, referrerStore{}, nil, nil, nil)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, svc)

				return
			}

			require.NoError(t, err)

			names := make([]string, 0, len(svc.tasks))
			for _, task := range svc.tasks {
				names = append(names, task.Name())
			}

			assert.Equal(t, tt.wantTasks, names)
		})
	}
}
