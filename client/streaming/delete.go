// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
)

// DeleteStream handles streaming delete operations in a self-contained manner.
// This follows the generator pattern from "Concurrency in Go" by Katherine Cox-Buday
// where functions take a context, input channel, and configuration, return an output channel,
// and manage their own goroutine lifecycle internally.
//
//nolint:cyclop // Streaming functions necessarily have high complexity due to concurrent patterns
func DeleteStream(ctx context.Context, client storetypes.StoreServiceClient, inStream <-chan *corev1.RecordRef, receiverFn ReceiverFn) (<-chan struct{}, error) {
	// Validate input parameters
	if receiverFn == nil {
		return nil, errors.New("receiver function is nil")
	}

	// Create gRPC stream once
	stream, err := client.Delete(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create delete stream: %w", err)
	}

	// Done channel
	doneCh := make(chan struct{})

	// Process items
	go func() {
		// Close stream and send notification once the goroutine ends
		defer stream.CloseSend()
		defer close(doneCh)

		// Process incoming record references
		for recordRef := range inStream {
			select {
			case <-ctx.Done():
				return
			default:
				// Send the record reference to the server
				err := stream.Send(recordRef)

				// Handle returned data using the receiver function.
				// If the receiver function returns an error, stop processing further items.
				if receiverErr := receiverFn(recordRef, err); receiverErr != nil {
					return
				}
			}
		}
	}()

	return doneCh, nil
}
