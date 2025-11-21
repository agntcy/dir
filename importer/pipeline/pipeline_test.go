// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"
	"errors"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// mockFetcher is a mock implementation of Fetcher for testing.
type mockFetcher struct {
	items []interface{}
	err   error
}

func (m *mockFetcher) Fetch(ctx context.Context) (<-chan interface{}, <-chan error) {
	dataCh := make(chan interface{})
	errCh := make(chan error, 1)

	go func() {
		defer close(dataCh)
		defer close(errCh)

		if m.err != nil {
			errCh <- m.err

			return
		}

		for _, item := range m.items {
			select {
			case dataCh <- item:
			case <-ctx.Done():
				return
			}
		}
	}()

	return dataCh, errCh
}

// mockTransformer is a mock implementation of Transformer for testing.
type mockTransformer struct {
	shouldFail bool
}

func (m *mockTransformer) Transform(ctx context.Context, source interface{}) (*corev1.Record, error) {
	if m.shouldFail {
		return nil, errors.New("transform failed")
	}

	// Create a simple record
	return &corev1.Record{}, nil
}

// mockPusher is a mock implementation of Pusher for testing.
type mockPusher struct {
	shouldFail bool
	pushed     []*corev1.Record
}

func (m *mockPusher) Push(ctx context.Context, inputCh <-chan *corev1.Record) (<-chan *corev1.RecordRef, <-chan error) {
	refCh := make(chan *corev1.RecordRef)
	errCh := make(chan error)

	go func() {
		defer close(refCh)
		defer close(errCh)

		// Consume all records from the input channel
		for record := range inputCh {
			m.pushed = append(m.pushed, record)

			if m.shouldFail {
				select {
				case errCh <- errors.New("push failed"):
				case <-ctx.Done():
					return
				}
			} else {
				// Send success response with a valid CID
				select {
				case refCh <- &corev1.RecordRef{Cid: "bafytest123"}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return refCh, errCh
}

func TestPipeline_Run_Success(t *testing.T) {
	ctx := context.Background()

	// Create mock stages
	fetcher := &mockFetcher{
		items: []interface{}{"item1", "item2", "item3"},
	}
	transformer := &mockTransformer{}
	pusher := &mockPusher{}

	// Create pipeline
	config := Config{
		TransformerWorkers: 2,
	}
	p := New(fetcher, transformer, pusher, config)

	// Run pipeline
	result, err := p.Run(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify results
	if result.TotalRecords != 3 {
		t.Errorf("expected 3 total records, got %d", result.TotalRecords)
	}

	if result.ImportedCount != 3 {
		t.Errorf("expected 3 imported records, got %d", result.ImportedCount)
	}

	if result.FailedCount != 0 {
		t.Errorf("expected 0 failed records, got %d", result.FailedCount)
	}

	if len(pusher.pushed) != 3 {
		t.Errorf("expected 3 pushed records, got %d", len(pusher.pushed))
	}
}

func TestDryRunPipeline_Run(t *testing.T) {
	ctx := context.Background()

	// Create mock stages
	fetcher := &mockFetcher{
		items: []interface{}{"item1", "item2"},
	}
	transformer := &mockTransformer{}

	// Create dry-run pipeline (no pusher)
	config := Config{
		TransformerWorkers: 2,
	}
	p := NewDryRun(fetcher, transformer, config)

	// Run pipeline
	result, err := p.Run(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify results - records should be counted but not pushed
	if result.TotalRecords != 2 {
		t.Errorf("expected 2 total records, got %d", result.TotalRecords)
	}

	if result.ImportedCount != 0 {
		t.Errorf("expected 0 imported records (dry-run), got %d", result.ImportedCount)
	}
}

func TestPipeline_Run_TransformError(t *testing.T) {
	ctx := context.Background()

	// Create mock stages with transformer that fails
	fetcher := &mockFetcher{
		items: []interface{}{"item1", "item2"},
	}
	transformer := &mockTransformer{shouldFail: true}
	pusher := &mockPusher{}

	// Create pipeline
	config := Config{
		TransformerWorkers: 2,
	}
	p := New(fetcher, transformer, pusher, config)

	// Run pipeline
	result, err := p.Run(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify results - all should fail transformation
	if result.TotalRecords != 2 {
		t.Errorf("expected 2 total records, got %d", result.TotalRecords)
	}

	if result.FailedCount != 2 {
		t.Errorf("expected 2 failed records, got %d", result.FailedCount)
	}

	if result.ImportedCount != 0 {
		t.Errorf("expected 0 imported records, got %d", result.ImportedCount)
	}

	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}

func TestPipeline_Run_PushError(t *testing.T) {
	ctx := context.Background()

	// Create mock stages with pusher that fails
	fetcher := &mockFetcher{
		items: []interface{}{"item1", "item2"},
	}
	transformer := &mockTransformer{}
	pusher := &mockPusher{shouldFail: true}

	// Create pipeline
	config := Config{
		TransformerWorkers: 2,
	}
	p := New(fetcher, transformer, pusher, config)

	// Run pipeline
	result, err := p.Run(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify results - all should fail push
	if result.TotalRecords != 2 {
		t.Errorf("expected 2 total records, got %d", result.TotalRecords)
	}

	if result.FailedCount != 2 {
		t.Errorf("expected 2 failed records, got %d", result.FailedCount)
	}

	if result.ImportedCount != 0 {
		t.Errorf("expected 0 imported records, got %d", result.ImportedCount)
	}

	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}

func TestPipeline_Run_FetchError(t *testing.T) {
	ctx := context.Background()

	// Create mock stages with fetcher that fails
	fetcher := &mockFetcher{
		err: errors.New("fetch failed"),
	}
	transformer := &mockTransformer{}
	pusher := &mockPusher{}

	// Create pipeline
	config := Config{
		TransformerWorkers: 2,
	}
	p := New(fetcher, transformer, pusher, config)

	// Run pipeline
	result, err := p.Run(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify results - should have fetch error
	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestPipeline_ConfigDefaults(t *testing.T) {
	fetcher := &mockFetcher{}
	transformer := &mockTransformer{}
	pusher := &mockPusher{}

	// Create pipeline with zero config values
	config := Config{}
	p := New(fetcher, transformer, pusher, config)

	// Verify defaults are set
	if p.config.TransformerWorkers != 5 {
		t.Errorf("expected default TransformerWorkers=5, got %d", p.config.TransformerWorkers)
	}
}
