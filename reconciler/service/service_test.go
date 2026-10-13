// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/reconciler/config"
	"github.com/agntcy/dir/reconciler/recordevents"
	"github.com/agntcy/dir/reconciler/tasks"
	"github.com/agntcy/dir/reconciler/tasks/identity"
	"github.com/agntcy/dir/reconciler/tasks/indexer"
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

	// block, when set, holds every run until it is closed or the run's context
	// ends.
	block chan struct{}
}

func (m *mockTask) Name() string            { return m.name }
func (m *mockTask) Interval() time.Duration { return m.interval }
func (m *mockTask) IsEnabled() bool         { return m.enabled }
func (m *mockTask) Run(ctx context.Context) error {
	m.runMu.Lock()
	m.runCalls++
	m.runMu.Unlock()

	if m.block != nil {
		select {
		case <-m.block:
		case <-ctx.Done():
		}
	}

	return m.runErr
}

func (m *mockTask) runs() int {
	m.runMu.Lock()
	defer m.runMu.Unlock()

	return m.runCalls
}

func newTestService() *Service {
	return &Service{
		tasks:   []tasks.Task{},
		wake:    make(map[string]chan struct{}),
		spacing: make(map[string]time.Duration),
		stopCh:  make(chan struct{}),
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

	t.Run("an ans block without the task registers nothing", func(t *testing.T) {
		s := newTestService()
		cfg := &config.Config{Identity: identity.Config{ANS: identity.ANSConfig{Enabled: true}}}
		require.NoError(t, s.registerIdentityTask(cfg, nil, referrerStore{}))
		assert.Empty(t, s.tasks)
	})

	t.Run("an invalid ans block fails", func(t *testing.T) {
		s := newTestService()
		cfg := &config.Config{Identity: identity.Config{Enabled: true, ANS: identity.ANSConfig{Enabled: true}}}
		require.ErrorContains(t, s.registerIdentityTask(cfg, nil, referrerStore{}), "failed to create identity task: configure the ans resolver")
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

// --- Trigger ---

// stopWhenDone stops the service at the end of a test.
func stopWhenDone(t *testing.T, s *Service) {
	t.Helper()

	t.Cleanup(func() { _ = s.Stop() })
}

func TestTrigger_RunsTheTaskNow(t *testing.T) {
	s := newTestService()
	task := &mockTask{name: "t", interval: time.Hour, enabled: true}
	s.addTask(task)

	require.NoError(t, s.Start(t.Context()))
	stopWhenDone(t, s)

	require.Eventually(t, func() bool { return task.runs() == 1 }, time.Second, time.Millisecond, "the task runs when it starts")

	s.Trigger("t")

	require.Eventually(t, func() bool { return task.runs() == 2 }, time.Second, time.Millisecond, "a trigger runs it without waiting for its interval")
}

func TestTrigger_IgnoresATaskThatIsNotRegistered(t *testing.T) {
	s := newTestService()

	assert.NotPanics(t, func() { s.Trigger("nobody") })
}

func TestTrigger_DoesNotRunADisabledTask(t *testing.T) {
	s := newTestService()
	task := &mockTask{name: "t", interval: time.Hour, enabled: false}
	s.addTask(task)

	require.NoError(t, s.Start(t.Context()))
	stopWhenDone(t, s)

	for range 5 {
		s.Trigger("t")
	}

	time.Sleep(50 * time.Millisecond)
	assert.Zero(t, task.runs())
}

// Asking again while the task is running is asking once: a burst of requests is
// one run after the one under way, not one run each.
func TestTrigger_CoalescesRequests(t *testing.T) {
	s := newTestService()
	task := &mockTask{name: "t", interval: time.Hour, enabled: true, block: make(chan struct{})}
	s.addTask(task)

	require.NoError(t, s.Start(t.Context()))
	stopWhenDone(t, s)

	require.Eventually(t, func() bool { return task.runs() == 1 }, time.Second, time.Millisecond)

	for range 50 {
		s.Trigger("t")
	}

	close(task.block)

	require.Eventually(t, func() bool { return task.runs() == 2 }, time.Second, time.Millisecond, "the requests made during the run are one more run")
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, task.runs(), "and no more than one")
}

// A task woken by events is not woken again until it has had a rest, so a
// stream of them cannot keep it running back to back.
func TestTrigger_SpacesTheRunsOfATaskWokenByEvents(t *testing.T) {
	s := newTestService()
	task := &mockTask{name: "t", interval: time.Hour, enabled: true}
	s.addTask(task)

	const spacing = 200 * time.Millisecond

	s.spacing["t"] = spacing

	require.NoError(t, s.Start(t.Context()))
	stopWhenDone(t, s)

	require.Eventually(t, func() bool { return task.runs() == 1 }, time.Second, time.Millisecond)

	asked := time.Now()

	s.Trigger("t")

	require.Eventually(t, func() bool { return task.runs() == 2 }, 2*time.Second, time.Millisecond)
	assert.GreaterOrEqual(t, time.Since(asked), spacing-50*time.Millisecond, "the run waited out the spacing since the last one ended")
}

func TestTrigger_StopEndsAWaitForTheSpacing(t *testing.T) {
	s := newTestService()
	task := &mockTask{name: "t", interval: time.Hour, enabled: true}
	s.addTask(task)

	s.spacing["t"] = time.Hour

	require.NoError(t, s.Start(t.Context()))

	require.Eventually(t, func() bool { return task.runs() == 1 }, time.Second, time.Millisecond)

	s.Trigger("t")

	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		_ = s.Stop()
	}()

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "Stop waited for the spacing to pass")
	}

	assert.Equal(t, 1, task.runs(), "the run that was waiting did not happen")
}

// --- WatchRecords ---

// fakeRecords reports the arrivals the test sends it, and one when it connects,
// as a source does.
type fakeRecords struct {
	pings   chan struct{}
	started chan struct{}
}

func newFakeRecords() *fakeRecords {
	return &fakeRecords{pings: make(chan struct{}, 16), started: make(chan struct{}, 1)}
}

func (f *fakeRecords) Listen(ctx context.Context, arrived func()) error {
	f.started <- struct{}{}

	arrived()

	for {
		select {
		case <-f.pings:
			arrived()
		case <-ctx.Done():
			return fmt.Errorf("stop: %w", ctx.Err())
		}
	}
}

func watchedService(t *testing.T, enabled bool, source recordevents.Source) (*Service, *mockTask) {
	t.Helper()

	s := newTestService()
	s.recordEvents = recordevents.Config{Enabled: enabled, Window: 30 * time.Millisecond}

	indexerTask := &mockTask{name: indexer.TaskName, interval: time.Hour, enabled: true}
	s.addTask(indexerTask)
	s.WatchRecords(source)

	return s, indexerTask
}

func TestWatchRecords_WakesTheIndexerWhenRecordsArrive(t *testing.T) {
	source := newFakeRecords()
	s, indexerTask := watchedService(t, true, source)

	assert.Equal(t, 30*time.Millisecond, s.spacing[indexer.TaskName], "an indexer woken by events is spaced by the window")

	require.NoError(t, s.Start(t.Context()))
	stopWhenDone(t, s)

	// It runs at the start, and again when the source connects, since records
	// may have arrived while no one listened.
	require.Eventually(t, func() bool { return indexerTask.runs() == 2 }, 2*time.Second, time.Millisecond)

	source.pings <- struct{}{}

	require.Eventually(t, func() bool { return indexerTask.runs() == 3 }, 2*time.Second, time.Millisecond, "a record arriving runs the indexer without waiting for its interval")
}

func TestWatchRecords_DoesNothingWhenEventsAreOff(t *testing.T) {
	source := newFakeRecords()
	s, indexerTask := watchedService(t, false, source)

	require.NoError(t, s.Start(t.Context()))
	stopWhenDone(t, s)

	require.Eventually(t, func() bool { return indexerTask.runs() == 1 }, time.Second, time.Millisecond)

	source.pings <- struct{}{}

	time.Sleep(150 * time.Millisecond)

	assert.Equal(t, 1, indexerTask.runs(), "the indexer runs at its interval only")
	assert.Empty(t, source.started, "nothing listens")
	assert.Empty(t, s.spacing, "and nothing is spaced")
}

func TestWatchRecords_WithoutASourceTheIndexerRunsAtItsInterval(t *testing.T) {
	s, indexerTask := watchedService(t, true, nil)

	require.NoError(t, s.Start(t.Context()))
	stopWhenDone(t, s)

	require.Eventually(t, func() bool { return indexerTask.runs() == 1 }, time.Second, time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, indexerTask.runs())
}

func TestWatchRecords_StopEndsTheWatch(t *testing.T) {
	source := newFakeRecords()
	s, _ := watchedService(t, true, source)

	require.NoError(t, s.Start(t.Context()))

	<-source.started

	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		_ = s.Stop()
	}()

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "Stop waited for the watch, which only ends with its context")
	}
}

// --- indexer to policy ---

// indexStore serves a record for any CID, and indexDB accepts every record.
type indexStore struct{ servertypes.StoreAPI }

func (indexStore) Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error) {
	return corev1.New(&typesv1alpha1.Record{Name: "agent", SchemaVersion: "0.7.0"}), nil
}

type indexDB struct{ servertypes.DatabaseAPI }

func (indexDB) AddRecord(coretypes.Record) error { return nil }

type oneTag struct{}

func (oneTag) Tags(_ context.Context, _ string, fn func([]string) error) error {
	return fn([]string{"baeareidp4vt6jw7tirdvk6qlcuqndobz24yejxovcjgcuv3qnnhwzqz4mi"})
}

func taskNamed(t *testing.T, s *Service, name string) tasks.Task {
	t.Helper()

	for _, task := range s.tasks {
		if task.Name() == name {
			return task
		}
	}

	require.FailNow(t, "task not registered", name)

	return nil
}

// What the indexer adds is what the policy task evaluates, so the policy task
// runs as soon as a run of the indexer has added something: not when its own
// interval comes round.
func TestRegisterTasks_IndexedRecordsWakeThePolicyTask(t *testing.T) {
	s := newTestService()

	cfg := &config.Config{
		Indexer:          indexer.Config{Enabled: true},
		PolicyEvaluation: policy.Config{Enabled: true},
	}

	require.NoError(t, s.registerTasks(cfg, indexDB{}, indexStore{}, oneTag{}, policyRegistry(t, "named"), nil))

	idx := taskNamed(t, s, indexer.TaskName)
	require.Empty(t, s.wake[policy.TaskName], "nothing has asked the policy task to run")

	require.NoError(t, idx.Run(t.Context()))

	assert.Len(t, s.wake[policy.TaskName], 1, "a run that indexed a record asks the policy task to run")
	assert.Empty(t, s.wake[indexer.TaskName], "and does not ask itself")

	// A run that finds nothing new asks for nothing.
	select {
	case <-s.wake[policy.TaskName]:
	default:
	}

	require.NoError(t, idx.Run(t.Context()))
	assert.Empty(t, s.wake[policy.TaskName])
}
