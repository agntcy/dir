// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
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

// Lookup retrieves metadata for a record using its reference.
func (c *Client) Lookup(ctx context.Context, recordRef *corev1.RecordRef) (*corev1.RecordMeta, error) {
	// Create streaming client
	stream, err := c.StoreServiceClient.Lookup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create lookup stream: %w", err)
	}

	// Send record reference
	if err := stream.Send(recordRef); err != nil {
		return nil, fmt.Errorf("failed to send record ref: %w", err)
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send stream: %w", err)
	}

	// Receive record metadata
	recordMeta, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive record metadata: %w", err)
	}

	return recordMeta, nil
}

// Delete removes a record from the store using its reference.
func (c *Client) Delete(ctx context.Context, recordRef *corev1.RecordRef) error {
	// Create streaming client
	stream, err := c.StoreServiceClient.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to create delete stream: %w", err)
	}

	// Send record reference
	if err := stream.Send(recordRef); err != nil {
		return fmt.Errorf("failed to send record ref: %w", err)
	}

	// Close send stream
	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("failed to close send stream: %w", err)
	}

	// For delete, we don't expect a response (just confirmation of completion)
	// The stream will close when the operation is complete
	return nil
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

// PushSignatureReferrer stores a signature using the PushReferrer RPC.
func (c *Client) PushSignatureReferrer(ctx context.Context, recordCID string, signature *signv1.Signature) error {
	// Create streaming client
	stream, err := c.StoreServiceClient.PushReferrer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create push referrer stream: %w", err)
	}

	// Create the push referrer request
	req := &storev1.PushReferrerRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordCID,
		},
		Options: &storev1.PushReferrerRequest_Signature{
			Signature: signature,
		},
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
