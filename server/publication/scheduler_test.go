// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publication

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	publypes "github.com/agntcy/dir/server/publication/types"
	"github.com/agntcy/dir/server/types"
)

// neverInterval is long enough that any sweep observed during a test must have
// come from the wake channel or from the initial sweep, never from the ticker.
const neverInterval = time.Hour

// fakeSchedulerDB serves a fixed set of pending publications and records the
// status transitions the scheduler applies.
type fakeSchedulerDB struct {
	types.PublicationDatabaseAPI

	mu       sync.Mutex
	pending  []types.PublicationObject
	inFlight []string

	// swept reports each sweep, letting a test order its actions against the
	// scheduler's startup sweep instead of racing it.
	swept chan struct{}
}

func (f *fakeSchedulerDB) GetPublicationsByStatus(routingv1.PublicationStatus) ([]types.PublicationObject, error) {
	f.mu.Lock()
	out := make([]types.PublicationObject, len(f.pending))
	copy(out, f.pending)
	f.mu.Unlock()

	select {
	case f.swept <- struct{}{}:
	default:
	}

	return out, nil
}

func (f *fakeSchedulerDB) UpdatePublicationStatus(publicationID string, _ routingv1.PublicationStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.inFlight = append(f.inFlight, publicationID)
	// A dispatched publication leaves the pending set, so later sweeps do not
	// redispatch it.
	f.pending = slices.DeleteFunc(f.pending, func(p types.PublicationObject) bool {
		return p.GetID() == publicationID
	})

	return nil
}

func (f *fakeSchedulerDB) setPending(ids ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.pending = nil
	for _, id := range ids {
		f.pending = append(f.pending, fakePublication{id: id})
	}
}

type fakePublication struct {
	types.PublicationObject

	id string
}

func (p fakePublication) GetID() string { return p.id }

// awaitWorkItem waits for a dispatch, failing rather than hanging if the
// scheduler never sweeps.
func awaitWorkItem(t *testing.T, queue <-chan publypes.WorkItem) publypes.WorkItem {
	t.Helper()

	select {
	case item := <-queue:
		return item
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not dispatch a publication")

		return publypes.WorkItem{}
	}
}

// awaitSweep waits for the scheduler to query for pending publications.
func awaitSweep(t *testing.T, db *fakeSchedulerDB) {
	t.Helper()

	select {
	case <-db.swept:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not sweep for pending publications")
	}
}

func TestWakeDispatchesWithoutWaitingForTheInterval(t *testing.T) {
	t.Parallel()

	db := &fakeSchedulerDB{swept: make(chan struct{}, 16)}
	queue := make(chan publypes.WorkItem, 8)
	wakeCh := make(chan struct{}, 1)

	scheduler := NewScheduler(db, queue, neverInterval, wakeCh)

	stopCh := make(chan struct{})
	defer close(stopCh)

	go scheduler.Run(t.Context(), stopCh)

	// Let the startup sweep observe an empty queue first. Without this the
	// publication below could be picked up by that sweep, and the test would
	// pass whether or not the wake is wired up.
	awaitSweep(t, db)

	// The publication now arrives while the service is running, so only a wake
	// can surface it before the hour is up.
	db.setPending("pub-1")

	wakeCh <- struct{}{}

	if got := awaitWorkItem(t, queue).PublicationID; got != "pub-1" {
		t.Fatalf("dispatched publication = %q, want %q", got, "pub-1")
	}
}

func TestSchedulerSweepsOnStart(t *testing.T) {
	t.Parallel()

	db := &fakeSchedulerDB{swept: make(chan struct{}, 16)}
	db.setPending("pub-existing")

	queue := make(chan publypes.WorkItem, 8)
	scheduler := NewScheduler(db, queue, neverInterval, make(chan struct{}, 1))

	stopCh := make(chan struct{})
	defer close(stopCh)

	go scheduler.Run(t.Context(), stopCh)

	if got := awaitWorkItem(t, queue).PublicationID; got != "pub-existing" {
		t.Fatalf("dispatched publication = %q, want %q", got, "pub-existing")
	}
}

func TestBacklogLargerThanTheQueueFullyDrains(t *testing.T) {
	t.Parallel()

	const (
		queueSize = 4
		backlog   = queueSize * 5
	)

	db := &fakeSchedulerDB{swept: make(chan struct{}, 16)}

	ids := make([]string, 0, backlog)
	for i := range backlog {
		ids = append(ids, fmt.Sprintf("pub-%d", i))
	}

	db.setPending(ids...)

	queue := make(chan publypes.WorkItem, queueSize)
	scheduler := NewScheduler(db, queue, neverInterval, make(chan struct{}, 1))

	stopCh := make(chan struct{})
	defer close(stopCh)

	go scheduler.Run(t.Context(), stopCh)

	// Consume as a worker would. A scheduler that dropped the overflow would
	// dispatch only queueSize items from this single sweep and then stall,
	// since neither the hour-long ticker nor a wake will fire.
	dispatched := make([]string, 0, backlog)
	for range backlog {
		dispatched = append(dispatched, awaitWorkItem(t, queue).PublicationID)
	}

	if !slices.Equal(dispatched, ids) {
		t.Fatalf("dispatched %d publications, want all %d in order", len(dispatched), backlog)
	}
}

func TestBacklogDispatchStopsOnShutdown(t *testing.T) {
	t.Parallel()

	db := &fakeSchedulerDB{swept: make(chan struct{}, 16)}
	db.setPending("pub-0", "pub-1", "pub-2")

	// An unbuffered queue with no consumer leaves the scheduler blocked mid
	// dispatch, which is where shutdown has to remain responsive.
	queue := make(chan publypes.WorkItem)
	scheduler := NewScheduler(db, queue, neverInterval, make(chan struct{}, 1))

	stopCh := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)

		scheduler.Run(t.Context(), stopCh)
	}()

	awaitSweep(t, db)
	close(stopCh)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not stop while blocked on a full queue")
	}
}

func TestWakeIsDroppedWhenOneIsAlreadyQueued(t *testing.T) {
	t.Parallel()

	svc := &Service{wakeCh: make(chan struct{}, 1)}

	// A burst of publishes against an unstarted scheduler must not block; the
	// single buffered slot absorbs them.
	for range 100 {
		svc.wake()
	}

	if len(svc.wakeCh) != 1 {
		t.Fatalf("queued wakes = %d, want 1", len(svc.wakeCh))
	}
}
