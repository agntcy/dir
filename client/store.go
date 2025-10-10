// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/streaming"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Push sends a complete record to the store and returns a record reference.
// The record must be ≤4MB as per the v1 store service specification.
func (c *Client) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	// Create streaming client
	stream, err := c.StoreServiceClient.Push(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create push stream: %w", err)
	}

	// Send complete record (up to 4MB)
	if err := stream.Send(record); err != nil {
		return nil, fmt.Errorf("failed to send record: %w", err)
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send stream: %w", err)
	}

	// Receive response for this record
	recordRef, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive record ref: %w", err)
	}

	return recordRef, nil
}

// PullStream retrieves multiple records efficiently using a single bidirectional stream.
// This method is ideal for batch operations and takes full advantage of gRPC streaming.
// The input channel allows you to send record refs as they become available.
//
// Returns two channels:
// - A channel of records as they are received from the server
// - A channel for errors (buffered, will contain at most one error)
//
// The method ensures proper stream lifecycle management by:
// - Running sender and receiver in separate goroutines for concurrent operation
// - Properly closing the send side after all refs are sent
// - Collecting and propagating any errors that occur
// - Respecting context cancellation
//
// Usage:
//
//	recordsCh, errCh := client.PullStream(ctx, refsCh)
//	for record := range recordsCh {
//	    // process record
//	}
//	if err := <-errCh; err != nil {
//	    // handle error
//	}
func (c *Client) PullStream(
	ctx context.Context,
	recordRefs <-chan *corev1.RecordRef,
) (<-chan *corev1.Record, <-chan error) {
	recordsCh := make(chan *corev1.Record)
	errCh := make(chan error, 1)

	// Validate inputs
	if ctx == nil {
		errCh <- errors.New("context is nil")

		close(recordsCh)
		close(errCh)

		return recordsCh, errCh
	}

	if recordRefs == nil {
		errCh <- errors.New("recordRefs channel is nil")

		close(recordsCh)
		close(errCh)

		return recordsCh, errCh
	}

	go func() {
		defer close(recordsCh)
		defer close(errCh)

		// Create streaming client
		stream, err := c.StoreServiceClient.Pull(ctx)
		if err != nil {
			errCh <- fmt.Errorf("failed to create pull stream: %w", err)

			return
		}

		// WaitGroup to coordinate sender and receiver goroutines
		var wg sync.WaitGroup

		// Error channel for internal goroutine communication
		internalErrCh := make(chan error, 2)

		// Sender goroutine: send all record refs
		wg.Add(1)

		go func() {
			defer wg.Done()

			for recordRef := range recordRefs {
				// Check context before sending to avoid unnecessary work
				select {
				case <-ctx.Done():
					internalErrCh <- ctx.Err()

					return
				default:
				}

				if err := stream.Send(recordRef); err != nil {
					internalErrCh <- fmt.Errorf("failed to send record ref: %w", err)

					return
				}
			}
			// Close the send side when done sending
			if err := stream.CloseSend(); err != nil {
				internalErrCh <- fmt.Errorf("failed to close send stream: %w", err)
			}
		}()

		// Receiver goroutine: receive all records
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				record, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					return
				}

				if err != nil {
					internalErrCh <- fmt.Errorf("failed to receive record: %w", err)

					return
				}

				// Validate received record
				if record == nil {
					internalErrCh <- errors.New("received nil record from stream")

					return
				}

				select {
				case recordsCh <- record:
				case <-ctx.Done():
					internalErrCh <- ctx.Err()

					return
				}
			}
		}()

		// Wait for both goroutines to complete
		wg.Wait()

		// Collect the first error if any occurred
		select {
		case err := <-internalErrCh:
			errCh <- err
			// Drain any remaining error to avoid goroutine leak
			select {
			case <-internalErrCh:
			default:
			}
		default:
		}
	}()

	return recordsCh, errCh
}

// Pull retrieves a single record from the store using its reference.
// This is a convenience wrapper around PullBatch for single-record operations.
func (c *Client) Pull(ctx context.Context, recordRef *corev1.RecordRef) (*corev1.Record, error) {
	records, err := c.PullBatch(ctx, []*corev1.RecordRef{recordRef})
	if err != nil {
		return nil, err
	}

	if len(records) != 1 {
		return nil, errors.New("no data returned")
	}

	return records[0], nil
}

// PullStreamWithCallback retrieves multiple records and processes each with the callback.
// This is the most efficient method for processing large batches of records,
// as it streams both input and output while processing records as they arrive.
//
// The method will return the first error encountered, either from the stream or from the callback.
// Processing stops immediately when an error occurs.
func (c *Client) PullStreamWithCallback(
	ctx context.Context,
	recordRefs <-chan *corev1.RecordRef,
	receiverFn streaming.ServerReceiverFn[corev1.RecordRef, corev1.Record],
) (<-chan struct{}, error) {
	stream, err := c.StoreServiceClient.Pull(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.NewServerStreamProcessor(ctx, stream, recordRefs, receiverFn)
}

// PullBatch retrieves multiple records in a single stream for efficiency.
// This is a convenience method that accepts a slice and returns a slice,
// built on top of the streaming implementation for consistency.
func (c *Client) PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error) {
	if len(recordRefs) == 0 {
		return nil, nil
	}

	// Convert slice to channel with context awareness
	refsCh := make(chan *corev1.RecordRef, len(recordRefs))
	go func() {
		defer close(refsCh)

		for _, ref := range recordRefs {
			select {
			case refsCh <- ref:
			case <-ctx.Done():
				// Context cancelled, stop sending
				return
			}
		}
	}()

	// Get streaming channels
	recordsCh, errCh := c.PullStream(ctx, refsCh)

	// Collect all records
	records := make([]*corev1.Record, 0, len(recordRefs))

	for {
		select {
		case record, ok := <-recordsCh:
			if !ok {
				// Stream closed, check for errors
				if err := <-errCh; err != nil {
					return nil, err
				}

				return records, nil
			}

			records = append(records, record)

		case err := <-errCh:
			return nil, err

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// PushBatch sends multiple records in a single stream for efficiency.
// This takes advantage of the streaming interface for batch operations.
//
//nolint:dupl // Similar structure to PullBatch but semantically different operations
func (c *Client) PushBatch(ctx context.Context, records []*corev1.Record) ([]*corev1.RecordRef, error) {
	if len(records) == 0 {
		return nil, nil
	}

	// Create streaming client
	stream, err := c.StoreServiceClient.Push(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create push stream: %w", err)
	}

	// Send all records
	for i, record := range records {
		if err := stream.Send(record); err != nil {
			return nil, fmt.Errorf("failed to send record %d: %w", i, err)
		}
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send stream: %w", err)
	}

	// Receive all responses
	var refs []*corev1.RecordRef //nolint:prealloc // We don't know the number of records in advance

	for i := range records {
		recordRef, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("failed to receive record ref %d: %w", i, err)
		}

		refs = append(refs, recordRef)
	}

	return refs, nil
}

// PushReferrer stores a signature using the PushReferrer RPC.
func (c *Client) PushReferrer(ctx context.Context, req *storev1.PushReferrerRequest) error {
	// Create streaming client
	stream, err := c.StoreServiceClient.PushReferrer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create push referrer stream: %w", err)
	}

	// Send the request
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("failed to send push referrer request: %w", err)
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("failed to close send stream: %w", err)
	}

	// Receive response
	_, err = stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive push referrer response: %w", err)
	}

	return nil
}

// PullReferrer retrieves all referrers using the PullReferrer RPC.
func (c *Client) PullReferrer(ctx context.Context, req *storev1.PullReferrerRequest) (<-chan *storev1.PullReferrerResponse, error) {
	// Create streaming client
	stream, err := c.StoreServiceClient.PullReferrer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull referrer stream: %w", err)
	}

	// Send the request
	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("failed to send pull referrer request: %w", err)
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send stream: %w", err)
	}

	resultCh := make(chan *storev1.PullReferrerResponse)

	go func() {
		defer close(resultCh)

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				logger.Error("failed to receive pull referrer response", "error", err)

				return
			}

			select {
			case resultCh <- response:
			case <-ctx.Done():
				logger.Error("context cancelled while receiving pull referrer response", "error", ctx.Err())

				return
			}
		}
	}()

	return resultCh, nil
}

// Lookup retrieves metadata for a record using its reference.
func (c *Client) Lookup(ctx context.Context, recordRef *corev1.RecordRef) (*corev1.RecordMeta, error) {
	resp, err := c.LookupBatch(ctx, []*corev1.RecordRef{recordRef})
	if err != nil {
		return nil, err
	}

	if len(resp) != 1 {
		return nil, errors.New("no data returned")
	}

	return resp[0], nil
}

// LookupBatch retrieves metadata for multiple records in a single stream for efficiency.
func (c *Client) LookupBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.RecordMeta, error) {
	if len(recordRefs) == 0 {
		return nil, nil
	}

	// Convert slice to channel with context awareness
	refChan := make(chan *corev1.RecordRef, len(recordRefs))
	go func() {
		defer close(refChan)

		for _, ref := range recordRefs {
			select {
			case refChan <- ref:
			case <-ctx.Done():
				// Context cancelled, stop sending
				return
			}
		}
	}()

	// Use channels to communicate results safely (no race conditions)
	metaCh := make(chan *corev1.RecordMeta, len(recordRefs))
	errCh := make(chan error, 1)

	doneCh, err := c.LookupStream(ctx, refChan, func(_ *corev1.RecordRef, meta *corev1.RecordMeta, err error) error {
		if err != nil {
			// Send error to channel (non-blocking, buffered channel)
			select {
			case errCh <- err:
			default:
			}

			return err
		}

		// Send metadata to channel (thread-safe)
		select {
		case metaCh <- meta:
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	<-doneCh
	close(metaCh)

	// Collect all metadata from channel (thread-safe)
	metas := make([]*corev1.RecordMeta, 0, len(recordRefs))
	for meta := range metaCh {
		metas = append(metas, meta)
	}

	// Check for errors
	select {
	case err := <-errCh:
		return metas, err
	default:
		return metas, nil
	}
}

// LookupStream provides efficient streaming lookup operations using channels.
// Record references are sent as they become available and metadata is returned as it's processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) LookupStream(ctx context.Context, refsCh <-chan *corev1.RecordRef, receiverFn streaming.ServerReceiverFn[corev1.RecordRef, corev1.RecordMeta]) (<-chan struct{}, error) {
	stream, err := c.StoreServiceClient.Lookup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create lookup stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.NewServerStreamProcessor(ctx, stream, refsCh, receiverFn)
}

// Delete removes a record from the store using its reference.
func (c *Client) Delete(ctx context.Context, recordRef *corev1.RecordRef) error {
	return c.DeleteBatch(ctx, []*corev1.RecordRef{recordRef})
}

// DeleteBatch removes multiple records from the store in a single stream for efficiency.
func (c *Client) DeleteBatch(ctx context.Context, recordRefs []*corev1.RecordRef) error {
	if len(recordRefs) == 0 {
		return nil
	}

	// Convert slice to channel with context awareness
	refChan := make(chan *corev1.RecordRef, len(recordRefs))
	go func() {
		defer close(refChan)

		for _, ref := range recordRefs {
			select {
			case refChan <- ref:
			case <-ctx.Done():
				// Context cancelled, stop sending
				return
			}
		}
	}()

	// Use channel to communicate error safely (no race condition)
	errCh := make(chan error, 1)

	doneCh, err := c.DeleteStream(ctx, refChan, func(_ *emptypb.Empty, err error) error {
		// Send error to channel (non-blocking, buffered channel)
		select {
		case errCh <- err:
		default:
		}

		return err
	})
	if err != nil {
		return err
	}

	<-doneCh

	// Retrieve error from channel (if any was sent)
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

// DeleteStream provides efficient streaming delete operations using channels.
// Record references are sent as they become available and delete confirmations are returned as they're processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) DeleteStream(ctx context.Context, refsCh <-chan *corev1.RecordRef, receiverFn streaming.ClientReceiverFn[emptypb.Empty]) (<-chan struct{}, error) {
	// Create gRPC stream once
	stream, err := c.StoreServiceClient.Delete(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.NewClientStreamProcessor(ctx, stream, refsCh, receiverFn)
}
