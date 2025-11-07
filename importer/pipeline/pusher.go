// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/config"
)

// ClientPusher is a Pusher implementation that uses the DIR client.
type ClientPusher struct {
	client config.ClientInterface
}

// NewClientPusher creates a new ClientPusher.
func NewClientPusher(client config.ClientInterface) *ClientPusher {
	return &ClientPusher{
		client: client,
	}
}

// Push sends records to DIR using the client.
//
// IMPLEMENTATION NOTE:
// This implementation pushes records sequentially (one-by-one) instead of using
// batch/streaming push. This is a temporary workaround because the current gRPC
// streaming implementation terminates the entire stream when a single record fails
// validation, preventing subsequent records from being processed.
//
// TODO: Switch back to streaming/batch push (PushStream) once the server-side
// implementation is updated to:
//  1. Return per-record error responses instead of terminating the stream
//  2. Allow the stream to continue processing remaining records after individual failures
//  3. This will require updating the proto to support a response type that can carry
//     either a RecordRef (success) or an error message (failure)
//
// The sequential approach ensures all records are attempted, even if some fail,
// at the cost of reduced throughput and increased latency.
func (p *ClientPusher) Push(ctx context.Context, inputCh <-chan *corev1.Record) (<-chan *corev1.RecordRef, <-chan error) {
	refCh := make(chan *corev1.RecordRef)
	errCh := make(chan error)

	go func() {
		defer close(refCh)
		defer close(errCh)

		// Push records one-by-one to ensure all records are processed
		// even if some fail validation
		for record := range inputCh {
			ref, err := p.client.Push(ctx, record)
			if err != nil {
				// Send error but continue processing remaining records
				select {
				case errCh <- err:
				case <-ctx.Done():
					return
				}

				continue
			}

			// Send successful reference
			select {
			case refCh <- ref:
			case <-ctx.Done():
				return
			}
		}
	}()

	return refCh, errCh
}
