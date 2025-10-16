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
	"github.com/agntcy/dir/client/streaming"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Push sends a complete record to the store and returns a record reference.
// This is a convenience wrapper around PushBatch for single-record operations.
// The record must be ≤4MB as per the v1 store service specification.
func (c *Client) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	refs, err := c.PushBatch(ctx, []*corev1.Record{record})
	if err != nil {
		return nil, err
	}

	if len(refs) != 1 {
		return nil, errors.New("no data returned")
	}

	return refs[0], nil
}

// PullStream retrieves multiple records efficiently using a single bidirectional stream.
// This method is ideal for batch operations and takes full advantage of gRPC streaming.
// The input channel allows you to send record refs as they become available.
func (c *Client) PullStream(ctx context.Context, refsCh <-chan *corev1.RecordRef) (streaming.StreamResult[corev1.Record], error) {
	stream, err := c.StoreServiceClient.Pull(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.ProcessBidiStream(ctx, stream, refsCh)
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

// PullBatch retrieves multiple records in a single stream for efficiency.
// This is a convenience method that accepts a slice and returns a slice,
// built on top of the streaming implementation for consistency.
func (c *Client) PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error) {
	// Use channel to communicate error safely (no race condition)
	result, err := c.PullStream(ctx, streaming.SliceToChan(ctx, recordRefs))
	if err != nil {
		return nil, err
	}

	// Check for results
	var errs error

	var metas []*corev1.Record

	for {
		select {
		case err := <-result.ErrCh():
			errs = errors.Join(errs, err)
		case resp := <-result.ResCh():
			metas = append(metas, resp)
		case <-result.DoneCh():
			return metas, errs
		}
	}
}

// PushStream uploads multiple records efficiently using a single bidirectional stream.
// This method is ideal for batch operations and takes full advantage of gRPC streaming.
// The input channel allows you to send records as they become available.
func (c *Client) PushStream(ctx context.Context, recordsCh <-chan *corev1.Record) (streaming.StreamResult[corev1.RecordRef], error) {
	stream, err := c.StoreServiceClient.Push(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create push stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.ProcessBidiStream(ctx, stream, recordsCh)
}

// PushBatch sends multiple records in a single stream for efficiency.
// This is a convenience method that accepts a slice and returns a slice,
// built on top of the streaming implementation for consistency.
func (c *Client) PushBatch(ctx context.Context, records []*corev1.Record) ([]*corev1.RecordRef, error) {
	// Use channel to communicate error safely (no race condition)
	result, err := c.PushStream(ctx, streaming.SliceToChan(ctx, records))
	if err != nil {
		return nil, err
	}

	// Check for results
	var errs error

	var refs []*corev1.RecordRef

	for {
		select {
		case err := <-result.ErrCh():
			errs = errors.Join(errs, err)
		case resp := <-result.ResCh():
			refs = append(refs, resp)
		case <-result.DoneCh():
			return refs, errs
		}
	}
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
	// Use channel to communicate error safely (no race condition)
	result, err := c.LookupStream(ctx, streaming.SliceToChan(ctx, recordRefs))
	if err != nil {
		return nil, err
	}

	// Check for results
	var errs error

	var metas []*corev1.RecordMeta

	for {
		select {
		case err := <-result.ErrCh():
			errs = errors.Join(errs, err)
		case resp := <-result.ResCh():
			metas = append(metas, resp)
		case <-result.DoneCh():
			return metas, errs
		}
	}
}

// LookupStream provides efficient streaming lookup operations using channels.
// Record references are sent as they become available and metadata is returned as it's processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
//
// Uses sequential streaming pattern (Send → Recv → Send → Recv) which ensures
// strict ordering of request-response pairs.
func (c *Client) LookupStream(ctx context.Context, refsCh <-chan *corev1.RecordRef) (streaming.StreamResult[corev1.RecordMeta], error) {
	stream, err := c.StoreServiceClient.Lookup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create lookup stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.ProcessBidiStream(ctx, stream, refsCh)
}

// Delete removes a record from the store using its reference.
func (c *Client) Delete(ctx context.Context, recordRef *corev1.RecordRef) error {
	return c.DeleteBatch(ctx, []*corev1.RecordRef{recordRef})
}

// DeleteBatch removes multiple records from the store in a single stream for efficiency.
func (c *Client) DeleteBatch(ctx context.Context, recordRefs []*corev1.RecordRef) error {
	// Use channel to communicate error safely (no race condition)
	result, err := c.DeleteStream(ctx, streaming.SliceToChan(ctx, recordRefs))
	if err != nil {
		return err
	}

	// Check for results
	for {
		select {
		case err := <-result.ErrCh():
			// If any error occurs, return immediately
			return err
		case <-result.ResCh():
			// We don't expect any results, just confirmations
		case <-result.DoneCh():
			return nil
		}
	}
}

// DeleteStream provides efficient streaming delete operations using channels.
// Record references are sent as they become available and delete confirmations are returned as they're processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) DeleteStream(ctx context.Context, refsCh <-chan *corev1.RecordRef) (streaming.StreamResult[emptypb.Empty], error) {
	// Create gRPC stream
	stream, err := c.StoreServiceClient.Delete(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.ProcessClientStream(ctx, stream, refsCh)
}
