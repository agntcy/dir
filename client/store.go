// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
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

// Pull retrieves a complete record from the store using its reference.
func (c *Client) Pull(ctx context.Context, recordRef *corev1.RecordRef) (*corev1.Record, error) {
	// Create streaming client
	stream, err := c.StoreServiceClient.Pull(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	// Send record reference
	if err := stream.Send(recordRef); err != nil {
		return nil, fmt.Errorf("failed to send record ref: %w", err)
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send stream: %w", err)
	}

	// Receive complete record
	record, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive record: %w", err)
	}

	return record, nil
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

// PullBatch retrieves multiple records in a single stream for efficiency.
//
//nolint:dupl // Similar structure to PushBatch but semantically different operations
func (c *Client) PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error) {
	if len(recordRefs) == 0 {
		return nil, nil
	}

	// Create streaming client
	stream, err := c.StoreServiceClient.Pull(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	// Send all record references
	for i, recordRef := range recordRefs {
		if err := stream.Send(recordRef); err != nil {
			return nil, fmt.Errorf("failed to send record ref %d: %w", i, err)
		}
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send stream: %w", err)
	}

	// Receive all records
	var records []*corev1.Record //nolint:prealloc // We don't know the number of records in advance

	for i := range recordRefs {
		record, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("failed to receive record %d: %w", i, err)
		}

		records = append(records, record)
	}

	return records, nil
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

	// Convert slice to channel
	refChan := make(chan *corev1.RecordRef, len(recordRefs))
	go func() {
		defer close(refChan)

		for _, ref := range recordRefs {
			refChan <- ref
		}
	}()

	// Use the self-contained streaming function
	var metas []*corev1.RecordMeta

	var firstErr error

	doneCh, err := c.LookupStream(ctx, refChan, func(_ *corev1.RecordRef, meta *corev1.RecordMeta, err error) error {
		if err != nil {
			firstErr = err // capture the first error

			return err
		}

		metas = append(metas, meta)

		return nil
	})
	if err != nil {
		return nil, err
	}

	<-doneCh

	return metas, firstErr
}

// LookupStream provides efficient streaming lookup operations using channels.
// Record references are sent as they become available and metadata is returned as it's processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) LookupStream(ctx context.Context, refsCh <-chan *corev1.RecordRef, receiverFn DataReceiverFn[*corev1.RecordMeta]) (<-chan struct{}, error) {
	// Validate input parameters
	if refsCh == nil {
		return nil, errors.New("refs channel is nil")
	}

	if receiverFn == nil {
		return nil, errors.New("receiver function is nil")
	}

	// Create gRPC stream once
	stream, err := c.StoreServiceClient.Lookup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create lookup stream: %w", err)
	}

	// Done channel
	doneCh := make(chan struct{})

	// Process items
	go func() {
		// Close stream and send notification once the goroutine ends
		//nolint:errcheck
		defer stream.CloseSend()
		defer close(doneCh)

		// Process incoming record references
		for {
			select {
			case <-ctx.Done():
				return
			case recordRef, ok := <-refsCh:
				// Exit if channel is closed
				if !ok {
					return
				}

				// Send the record reference to the network buffer
				// Handle any error using the receiver function
				err := stream.Send(recordRef)
				if err != nil {
					if recvErr := receiverFn(recordRef, nil, err); recvErr != nil {
						return
					}

					continue
				}

				// Receive the response from the network buffer
				// Handle the response and any error using the receiver function
				response, err := stream.Recv()
				if recvErr := receiverFn(recordRef, response, err); recvErr != nil {
					return
				}
			}
		}
	}()

	return doneCh, nil
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

	// Convert slice to channel
	refChan := make(chan *corev1.RecordRef, len(recordRefs))
	go func() {
		defer close(refChan)

		for _, ref := range recordRefs {
			refChan <- ref
		}
	}()

	// Use the self-contained streaming function
	var firstErr error

	doneCh, err := c.DeleteStream(ctx, refChan, func(err error) error {
		firstErr = err

		return err
	})
	if err != nil {
		return err
	}

	<-doneCh

	return firstErr
}

// DeleteStream provides efficient streaming delete operations using channels.
// Record references are sent as they become available and delete confirmations are returned as they're processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) DeleteStream(ctx context.Context, refsCh <-chan *corev1.RecordRef, receiverFn ErrReceiverFn) (<-chan struct{}, error) {
	// Validate input parameters
	if refsCh == nil {
		return nil, errors.New("refs channel is nil")
	}

	if receiverFn == nil {
		return nil, errors.New("receiver function is nil")
	}

	// Create gRPC stream once
	stream, err := c.StoreServiceClient.Delete(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete stream: %w", err)
	}

	// Done channel
	doneCh := make(chan struct{})

	// Process items
	go func() {
		// Close stream and send notification once the goroutine ends
		//nolint:errcheck
		defer stream.CloseSend()
		defer close(doneCh)

		// Process incoming record references
		for {
			select {
			case <-ctx.Done():
				return
			case recordRef, ok := <-refsCh:
				// Once the channel is closed, send the data through the stream and exit.
				// Handle any error using the receiver function.
				if !ok {
					_, err := stream.CloseAndRecv()
					_ = receiverFn(err)

					return
				}

				// Send the record reference to the network buffer
				_ = stream.Send(recordRef)
			}
		}
	}()

	return doneCh, nil
}
