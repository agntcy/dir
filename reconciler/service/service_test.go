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
	"github.com/agntcy/dir/reconciler/tasks/policy"
	servertypes "github.com/agntcy/dir/server/types"
	recordvalidators "github.com/agntcy/dir/server/validators"
	validatorsconfig "github.com/agntcy/dir/server/validators/config"
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

// policyRegistry builds a validator registry with one CEL policy, named name.
func policyRegistry(t *testing.T, name string) *recordvalidators.Registry {
	t.Helper()

	registry, err := recordvalidators.NewRegistry(t.Context(), validatorsconfig.Config{{
		Provider: validatorsconfig.ProviderCEL,
		Ops:      []string{validatorsconfig.OpEvaluate},
		Config: map[string]any{
			validatorsconfig.ConfigKeyName:        name,
			validatorsconfig.ConfigKeyExpressions: []string{`record.name != ""`},
		},
	}}, "")
	require.NoError(t, err)

	return registry
}

func TestRegisterPolicyTask_Disabled(t *testing.T) {
	s := newTestService()

	require.NoError(t, s.registerPolicyTask(policy.Config{}, nil, nil, nil))
	assert.Empty(t, s.tasks)

	// Policies with nothing to evaluate them are reported, not registered.
	require.NoError(t, s.registerPolicyTask(policy.Config{}, nil, nil, policyRegistry(t, "named")))
	assert.Empty(t, s.tasks)
}

// With no policy configured, enabling the task registers it disabled, so
// policy_evaluation.enabled has a visible effect instead of none.
func TestRegisterPolicyTask_EnabledWithoutPolicies(t *testing.T) {
	s := newTestService()

	require.NoError(t, s.registerPolicyTask(policy.Config{Enabled: true}, nil, nil, nil))
	require.Len(t, s.tasks, 1)
	assert.Equal(t, "policy", s.tasks[0].Name())
	assert.False(t, s.tasks[0].IsEnabled())
}

// Each validator with op evaluate is a policy the task evaluates.
func TestRegisterPolicyTask_EnabledWithPolicies(t *testing.T) {
	s := newTestService()

	require.NoError(t, s.registerPolicyTask(policy.Config{Enabled: true}, nil, nil, policyRegistry(t, "named")))
	require.Len(t, s.tasks, 1)
	assert.Equal(t, "policy", s.tasks[0].Name())
	assert.True(t, s.tasks[0].IsEnabled())
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

// plainStore is a store with no referrer support.
type plainStore struct{ servertypes.StoreAPI }

// referrerStore is a store that also supports referrers.
type referrerStore struct {
	servertypes.StoreAPI
	servertypes.ReferrerStoreAPI
}

func TestRegisterIdentityTask(t *testing.T) {
	enabled := &config.Config{Identity: identity.Config{Enabled: true}}

	t.Run("disabled registers nothing", func(t *testing.T) {
		s := newTestService()
		require.NoError(t, s.registerIdentityTask(&config.Config{}, nil, referrerStore{}))
		assert.Empty(t, s.tasks)
	})

	t.Run("enabled registers the task", func(t *testing.T) {
		s := newTestService()
		require.NoError(t, s.registerIdentityTask(enabled, nil, referrerStore{}))

		require.Len(t, s.tasks, 1)
		assert.Equal(t, "identity", s.tasks[0].Name())
		assert.True(t, s.tasks[0].IsEnabled())
	})

	t.Run("a store without referrers skips it", func(t *testing.T) {
		s := newTestService()
		require.NoError(t, s.registerIdentityTask(enabled, nil, plainStore{}))
		assert.Empty(t, s.tasks)
	})
}

// The task is only part of a service the configuration asks for it in.
func TestRegisterTasks_IdentityIsOffByDefault(t *testing.T) {
	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	cfg.Regsync.Enabled, cfg.Indexer.Enabled, cfg.Signature.Enabled, cfg.Metrics.Enabled = false, false, false, false

	s := newTestService()
	require.NoError(t, s.registerTasks(cfg, nil, referrerStore{}, nil, nil, nil))
	assert.Empty(t, s.tasks)

	t.Setenv("RECONCILER_IDENTITY_ENABLED", "true")

	cfg, err = config.LoadConfig()
	require.NoError(t, err)

	cfg.Regsync.Enabled, cfg.Indexer.Enabled, cfg.Signature.Enabled, cfg.Metrics.Enabled = false, false, false, false

	s = newTestService()
	require.NoError(t, s.registerTasks(cfg, nil, referrerStore{}, nil, nil, nil))
	require.Len(t, s.tasks, 1)
	assert.Equal(t, "identity", s.tasks[0].Name())
}
