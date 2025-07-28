// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/client/streaming"
)

// PushResult represents the result of a push operation.
// This is an alias to the streaming package's PushResult for clean API exposure.
type PushResult = streaming.PushResult

// PullResult represents the result of a pull operation.
// This is an alias to the streaming package's PullResult for clean API exposure.
type PullResult = streaming.PullResult

// LookupResult represents the result of a lookup operation.
// This is an alias to the streaming package's LookupResult for clean API exposure.
type LookupResult = streaming.LookupResult

// DeleteResult represents the result of a delete operation.
// This is an alias to the streaming package's DeleteResult for clean API exposure.
type DeleteResult = streaming.DeleteResult

// Push sends a complete record to the store and returns a record reference.
// The record must be ≤4MB as per the v1alpha2 store service specification.
func (c *Client) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	// Convert single record to channel
	records := make(chan *corev1.Record, 1)
	records <- record
	close(records)

	// Use the self-contained streaming function
	results := streaming.PushStream(ctx, records, c.StoreServiceClient)
	result := <-results

	return result.RecordRef, result.Error
}

// PushStream provides efficient streaming push operations using channels.
// Records are sent as they become available and results are returned as they're processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) PushStream(ctx context.Context, records <-chan *corev1.Record) <-chan PushResult {
	return streaming.PushStream(ctx, records, c.StoreServiceClient)
}

// Pull retrieves a complete record from the store using its reference.
func (c *Client) Pull(ctx context.Context, recordRef *corev1.RecordRef) (*corev1.Record, error) {
	// Convert single record ref to channel
	refs := make(chan *corev1.RecordRef, 1)
	refs <- recordRef
	close(refs)

	// Use the self-contained streaming function
	results := streaming.PullStream(ctx, refs, c.StoreServiceClient)
	result := <-results

	return result.Record, result.Error
}

// PullStream provides efficient streaming pull operations using channels.
// Record references are sent as they become available and records are returned as they're processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) PullStream(ctx context.Context, refs <-chan *corev1.RecordRef) <-chan PullResult {
	return streaming.PullStream(ctx, refs, c.StoreServiceClient)
}

// Lookup retrieves metadata for a record using its reference.
func (c *Client) Lookup(ctx context.Context, recordRef *corev1.RecordRef) (*corev1.RecordMeta, error) {
	// Convert single record ref to channel
	refs := make(chan *corev1.RecordRef, 1)
	refs <- recordRef
	close(refs)

	// Use the self-contained streaming function
	results := streaming.LookupStream(ctx, refs, c.StoreServiceClient)
	result := <-results

	return result.RecordMeta, result.Error
}

// LookupStream provides efficient streaming lookup operations using channels.
// Record references are sent as they become available and metadata is returned as it's processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) LookupStream(ctx context.Context, refs <-chan *corev1.RecordRef) <-chan LookupResult {
	return streaming.LookupStream(ctx, refs, c.StoreServiceClient)
}

// Delete removes a record from the store using its reference.
func (c *Client) Delete(ctx context.Context, recordRef *corev1.RecordRef) error {
	// Convert single record ref to channel
	refs := make(chan *corev1.RecordRef, 1)
	refs <- recordRef
	close(refs)

	// Use the self-contained streaming function
	results := streaming.DeleteStream(ctx, refs, c.StoreServiceClient)
	result := <-results

	return result.Error
}

// DeleteStream provides efficient streaming delete operations using channels.
// Record references are sent as they become available and delete confirmations are returned as they're processed.
// This method maintains a single gRPC stream for all operations, dramatically improving efficiency.
func (c *Client) DeleteStream(ctx context.Context, refs <-chan *corev1.RecordRef) <-chan DeleteResult {
	return streaming.DeleteStream(ctx, refs, c.StoreServiceClient)
}

// PushBatch sends multiple records in a single stream for efficiency.
// This takes advantage of the streaming interface for batch operations.
func (c *Client) PushBatch(ctx context.Context, records []*corev1.Record) ([]*corev1.RecordRef, error) {
	if len(records) == 0 {
		return nil, nil
	}

	// Convert slice to channel
	recordChan := make(chan *corev1.Record, len(records))
	go func() {
		defer close(recordChan)

		for _, record := range records {
			recordChan <- record
		}
	}()

	// Use the streaming function
	results := c.PushStream(ctx, recordChan)

	var refs []*corev1.RecordRef

	var firstError error

	for result := range results {
		if result.Error != nil && firstError == nil {
			firstError = result.Error
		}

		if result.RecordRef != nil {
			refs = append(refs, result.RecordRef)
		}
	}

	return refs, firstError
}

// PullBatch retrieves multiple records in a single stream for efficiency.
func (c *Client) PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error) {
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

	// Use the streaming function
	results := c.PullStream(ctx, refChan)

	var records []*corev1.Record

	var firstError error

	for result := range results {
		if result.Error != nil && firstError == nil {
			firstError = result.Error
		}

		if result.Record != nil {
			records = append(records, result.Record)
		}
	}

	return records, firstError
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

	// Use the streaming function
	results := c.LookupStream(ctx, refChan)

	var metas []*corev1.RecordMeta

	var firstError error

	for result := range results {
		if result.Error != nil && firstError == nil {
			firstError = result.Error
		}

		if result.RecordMeta != nil {
			metas = append(metas, result.RecordMeta)
		}
	}

	return metas, firstError
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

	// Use the streaming function
	results := c.DeleteStream(ctx, refChan)

	// Return first error encountered
	for result := range results {
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}
