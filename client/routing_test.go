// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"io"
	"runtime"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"google.golang.org/grpc"
)

const (
	// Test data constants.
	testRecordCID = "test-cid"

	// Test size constants.
	testResponseCountSmall  = 10
	testResponseCountMedium = 100
	testResponseCountLarge  = 1000

	// Test timeout constants.
	testSlowServerDelay       = 10 * time.Millisecond
	testFastServerDelay       = 1 * time.Millisecond
	testMediumServerDelay     = 50 * time.Millisecond
	testResponseTimeout       = 1 * time.Second
	testDrainTimeout          = 500 * time.Millisecond
	testCleanupDelay          = 50 * time.Millisecond
	testLongCleanupDelay      = 200 * time.Millisecond
	testBenchmarkCleanupDelay = 100 * time.Millisecond

	// Goroutine leak tolerance.
	testGoroutineLeakTolerance          = 2
	testBenchmarkGoroutineLeakTolerance = 10

	// Test read counts.
	testPartialReadCount = 5
	testSmallReadCount   = 10
)

// ============================================================================
// Issue 5: Blocked Goroutine Leaks Tests
// ============================================================================

// mockListStream simulates a gRPC List stream.
type mockListStream struct {
	responses []*routingv1.ListResponse
	index     int
	delay     time.Duration // Delay between sends to simulate slow server
	grpc.ClientStream
}

func (m *mockListStream) Recv() (*routingv1.ListResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.index >= len(m.responses) {
		return nil, io.EOF
	}

	resp := m.responses[m.index]
	m.index++

	return resp, nil
}

// mockSearchStream simulates a gRPC Search stream.
type mockSearchStream struct {
	responses []*routingv1.SearchResponse
	index     int
	delay     time.Duration
	grpc.ClientStream
}

func (m *mockSearchStream) Recv() (*routingv1.SearchResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.index >= len(m.responses) {
		return nil, io.EOF
	}

	resp := m.responses[m.index]
	m.index++

	return resp, nil
}

// mockRoutingServiceClient is a mock for testing routing methods.
type mockRoutingServiceClient struct {
	listResponses   []*routingv1.ListResponse
	searchResponses []*routingv1.SearchResponse
	listDelay       time.Duration
	searchDelay     time.Duration
	routingv1.RoutingServiceClient
}

func (m *mockRoutingServiceClient) List(ctx context.Context, req *routingv1.ListRequest, opts ...grpc.CallOption) (routingv1.RoutingService_ListClient, error) {
	return &mockListStream{
		responses: m.listResponses,
		delay:     m.listDelay,
	}, nil
}

func (m *mockRoutingServiceClient) Search(ctx context.Context, req *routingv1.SearchRequest, opts ...grpc.CallOption) (routingv1.RoutingService_SearchClient, error) {
	return &mockSearchStream{
		responses: m.searchResponses,
		delay:     m.searchDelay,
	}, nil
}

// countGoroutines returns the current number of goroutines.
func countGoroutines() int {
	return runtime.NumGoroutine()
}

// testContextCancellation is a helper that tests context cancellation for streaming methods.
func testContextCancellation(t *testing.T, startStream func(context.Context) (<-chan interface{}, error), name string) {
	t.Helper()

	initialGoroutines := countGoroutines()
	t.Logf("Initial goroutines: %d", initialGoroutines)

	ctx, cancel := context.WithCancel(context.Background())

	resCh, err := startStream(ctx)
	if err != nil {
		t.Fatalf("%s failed: %v", name, err)
	}

	// Read a few responses
	readCount := 0

	for range testPartialReadCount {
		select {
		case _, ok := <-resCh:
			if !ok {
				t.Fatal("Channel closed unexpectedly")
			}

			readCount++
		case <-time.After(testResponseTimeout):
			t.Fatal("Timeout waiting for response")
		}
	}

	t.Logf("Read %d responses", readCount)

	// Cancel context (consumer stops reading)
	cancel()

	// Drain remaining responses to allow goroutine to exit
	drained := 0
	drainTimeout := time.After(testDrainTimeout)

drainLoop:
	for {
		select {
		case _, ok := <-resCh:
			if !ok {
				// Channel closed, good!
				break drainLoop
			}

			drained++
		case <-drainTimeout:
			t.Log("Timeout while draining channel")

			break drainLoop
		}
	}

	t.Logf("Drained %d additional responses", drained)

	// Wait for goroutine to clean up
	time.Sleep(testCleanupDelay)

	// Count goroutines after
	finalGoroutines := countGoroutines()
	t.Logf("Final goroutines: %d", finalGoroutines)

	// Verify no goroutine leak (allow some tolerance for test framework goroutines)
	if finalGoroutines > initialGoroutines+testGoroutineLeakTolerance {
		t.Errorf("Goroutine leak detected: initial=%d, final=%d, leaked=%d",
			initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
	}
}

// testConsumerStopsReading is a helper that tests consumer stopping reading for streaming methods.
func testConsumerStopsReading(t *testing.T, startStream func(context.Context) (<-chan interface{}, error), name string) {
	t.Helper()

	initialGoroutines := countGoroutines()
	t.Logf("Initial goroutines: %d", initialGoroutines)

	ctx, cancel := context.WithTimeout(context.Background(), testContextTimeout)
	defer cancel()

	resCh, err := startStream(ctx)
	if err != nil {
		t.Fatalf("%s failed: %v", name, err)
	}

	// Read only a few responses and stop (consumer stops reading)
	readCount := 0

	for range testSmallReadCount {
		<-resCh

		readCount++
	}

	t.Logf("Read %d responses, then stopped", readCount)

	// Cancel context to signal we're done
	cancel()

	// Wait for cleanup
	time.Sleep(testLongCleanupDelay)

	finalGoroutines := countGoroutines()
	t.Logf("Final goroutines: %d", finalGoroutines)

	// Verify no significant goroutine leak
	if finalGoroutines > initialGoroutines+testGoroutineLeakTolerance {
		t.Errorf("Goroutine leak detected: initial=%d, final=%d, leaked=%d",
			initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
	}
}

// TestList_ContextCancellation tests that List() properly handles context cancellation.
func TestList_ContextCancellation(t *testing.T) {
	// Create mock responses
	responses := make([]*routingv1.ListResponse, testResponseCountMedium)
	for i := range testResponseCountMedium {
		responses[i] = &routingv1.ListResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		listResponses: responses,
		listDelay:     testSlowServerDelay,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	// Use helper to test context cancellation
	testContextCancellation(t, func(ctx context.Context) (<-chan interface{}, error) {
		ch, err := client.List(ctx, &routingv1.ListRequest{})
		if err != nil {
			return nil, err
		}
		// Convert typed channel to interface{} channel
		outCh := make(chan interface{})

		go func() {
			defer close(outCh)

			for v := range ch {
				outCh <- v
			}
		}()

		return outCh, nil
	}, "List()")
}

// TestList_ConsumerStopsReading tests that List() handles consumer stopping reading.
func TestList_ConsumerStopsReading(t *testing.T) {
	// Create many mock responses
	responses := make([]*routingv1.ListResponse, testResponseCountLarge)
	for i := range testResponseCountLarge {
		responses[i] = &routingv1.ListResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		listResponses: responses,
		listDelay:     testFastServerDelay,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	// Use helper to test consumer stops reading
	testConsumerStopsReading(t, func(ctx context.Context) (<-chan interface{}, error) {
		ch, err := client.List(ctx, &routingv1.ListRequest{})
		if err != nil {
			return nil, err
		}
		// Convert typed channel to interface{} channel
		outCh := make(chan interface{})

		go func() {
			defer close(outCh)

			for v := range ch {
				outCh <- v
			}
		}()

		return outCh, nil
	}, "List()")
}

// TestList_FullConsumption tests that List() works correctly when consumer reads everything.
func TestList_FullConsumption(t *testing.T) {
	responses := make([]*routingv1.ListResponse, testResponseCountSmall)
	for i := range testResponseCountSmall {
		responses[i] = &routingv1.ListResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		listResponses: responses,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	ctx := context.Background()

	resCh, err := client.List(ctx, &routingv1.ListRequest{})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Read all responses
	count := 0
	for range resCh {
		count++
	}

	if count != len(responses) {
		t.Errorf("Expected to receive %d responses, got %d", len(responses), count)
	}
}

// TestSearchRouting_ContextCancellation tests that SearchRouting() properly handles context cancellation.
func TestSearchRouting_ContextCancellation(t *testing.T) {
	responses := make([]*routingv1.SearchResponse, testResponseCountMedium)
	for i := range testResponseCountMedium {
		responses[i] = &routingv1.SearchResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		searchResponses: responses,
		searchDelay:     testSlowServerDelay,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	// Use helper to test context cancellation
	testContextCancellation(t, func(ctx context.Context) (<-chan interface{}, error) {
		ch, err := client.SearchRouting(ctx, &routingv1.SearchRequest{})
		if err != nil {
			return nil, err
		}
		// Convert typed channel to interface{} channel
		outCh := make(chan interface{})

		go func() {
			defer close(outCh)

			for v := range ch {
				outCh <- v
			}
		}()

		return outCh, nil
	}, "SearchRouting()")
}

// TestSearchRouting_ConsumerStopsReading tests that SearchRouting() handles consumer stopping reading.
func TestSearchRouting_ConsumerStopsReading(t *testing.T) {
	responses := make([]*routingv1.SearchResponse, testResponseCountLarge)
	for i := range testResponseCountLarge {
		responses[i] = &routingv1.SearchResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		searchResponses: responses,
		searchDelay:     testFastServerDelay,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	// Use helper to test consumer stops reading
	testConsumerStopsReading(t, func(ctx context.Context) (<-chan interface{}, error) {
		ch, err := client.SearchRouting(ctx, &routingv1.SearchRequest{})
		if err != nil {
			return nil, err
		}
		// Convert typed channel to interface{} channel
		outCh := make(chan interface{})

		go func() {
			defer close(outCh)

			for v := range ch {
				outCh <- v
			}
		}()

		return outCh, nil
	}, "SearchRouting()")
}

// TestSearchRouting_FullConsumption tests that SearchRouting() works correctly when consumer reads everything.
func TestSearchRouting_FullConsumption(t *testing.T) {
	responses := make([]*routingv1.SearchResponse, testResponseCountSmall)
	for i := range testResponseCountSmall {
		responses[i] = &routingv1.SearchResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		searchResponses: responses,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	ctx := context.Background()

	resCh, err := client.SearchRouting(ctx, &routingv1.SearchRequest{})
	if err != nil {
		t.Fatalf("SearchRouting() failed: %v", err)
	}

	// Read all responses
	count := 0
	for range resCh {
		count++
	}

	if count != len(responses) {
		t.Errorf("Expected to receive %d responses, got %d", len(responses), count)
	}
}

// TestList_ImmediateCancellation tests List() with immediate context cancellation.
func TestList_ImmediateCancellation(t *testing.T) {
	responses := make([]*routingv1.ListResponse, testResponseCountMedium)
	for i := range testResponseCountMedium {
		responses[i] = &routingv1.ListResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		listResponses: responses,
		listDelay:     testMediumServerDelay,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resCh, err := client.List(ctx, &routingv1.ListRequest{})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Channel should close quickly due to cancelled context
	select {
	case _, ok := <-resCh:
		if ok {
			// If we got a response, that's OK - might have sent before cancel was processed
			t.Logf("Got response before cancellation was processed")
		}
	case <-time.After(testDrainTimeout):
		t.Error("Channel should close when context is already cancelled")
	}

	// Wait for cleanup
	time.Sleep(testBenchmarkCleanupDelay)
}

// TestSearchRouting_ImmediateCancellation tests SearchRouting() with immediate context cancellation.
func TestSearchRouting_ImmediateCancellation(t *testing.T) {
	responses := make([]*routingv1.SearchResponse, testResponseCountMedium)
	for i := range testResponseCountMedium {
		responses[i] = &routingv1.SearchResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		searchResponses: responses,
		searchDelay:     testMediumServerDelay,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resCh, err := client.SearchRouting(ctx, &routingv1.SearchRequest{})
	if err != nil {
		t.Fatalf("SearchRouting() failed: %v", err)
	}

	// Channel should close quickly
	select {
	case _, ok := <-resCh:
		if ok {
			t.Logf("Got response before cancellation was processed")
		}
	case <-time.After(testDrainTimeout):
		t.Error("Channel should close when context is already cancelled")
	}

	// Wait for cleanup
	time.Sleep(testBenchmarkCleanupDelay)
}

// BenchmarkList_NoLeak benchmarks List() to detect goroutine leaks under load.
func BenchmarkList_NoLeak(b *testing.B) {
	responses := make([]*routingv1.ListResponse, testResponseCountSmall)
	for i := range testResponseCountSmall {
		responses[i] = &routingv1.ListResponse{
			RecordRef: &corev1.RecordRef{
				Cid: testRecordCID,
			},
		}
	}

	mockClient := &mockRoutingServiceClient{
		listResponses: responses,
	}

	client := &Client{
		RoutingServiceClient: mockClient,
	}

	initialGoroutines := countGoroutines()

	b.ResetTimer()

	for range b.N {
		ctx, cancel := context.WithCancel(context.Background())

		resCh, err := client.List(ctx, &routingv1.ListRequest{})
		if err != nil {
			b.Fatalf("List() failed: %v", err)
		}

		// Read a few then cancel
		for range testPartialReadCount - 2 {
			<-resCh
		}

		cancel()

		// Drain channel
		for range resCh {
		}
	}

	b.StopTimer()

	// Check for goroutine leaks
	runtime.GC()
	time.Sleep(testBenchmarkCleanupDelay)

	finalGoroutines := countGoroutines()

	if finalGoroutines > initialGoroutines+testBenchmarkGoroutineLeakTolerance {
		b.Errorf("Potential goroutine leak: initial=%d, final=%d, leaked=%d",
			initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
	}
}
