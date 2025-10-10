// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"
)

// NewSequentialStreamProcessor handles sequential bidirectional streaming.
//
// Pattern: Send → Recv → Send → Recv (request-response pairs)
//
// This processor implements a synchronous request-response pattern over a bidirectional stream.
// For each input, it:
// 1. Sends the input to the server
// 2. Waits for the corresponding response
// 3. Calls the receiver function with both input and output
// 4. Repeats for the next input
//
// This is useful when:
// - Server responds immediately to each request
// - Order must be strictly preserved
// - Simple synchronous processing is needed
// - Each request-response pair is independent
//
// Note: This pattern does not maximize throughput. For high-performance scenarios
// with many items, consider using NewBidirectionalStreamProcessor instead.
//
// Returns a done channel that closes when processing completes, and an error if validation fails.
func NewSequentialStreamProcessor[InT, OutT any](ctx context.Context, stream BidirectionalStream[InT, OutT], refsCh <-chan *InT, receiverFn SequentialReceiverFn[InT, OutT]) (<-chan struct{}, error) {
	// Validate input parameters
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	if stream == nil {
		return nil, errors.New("stream is nil")
	}

	if refsCh == nil {
		return nil, errors.New("refs channel is nil")
	}

	if receiverFn == nil {
		return nil, errors.New("receiver function is nil")
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

				// Check context again before potentially blocking Send operation
				select {
				case <-ctx.Done():
					return
				default:
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
