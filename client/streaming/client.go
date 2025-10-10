// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"
)

// NewClientStreamProcessor handles client streaming pattern (many inputs → one output).
//
// Pattern: Send → Send → Send → CloseAndRecv()
//
// This processor is ideal for operations where multiple requests are sent to the server,
// and a single final response is received after all requests have been processed.
//
// The processor:
// - Sends all inputs from the channel to the stream
// - Closes the send side when input channel closes
// - Receives the final response via CloseAndRecv()
// - Calls the receiver function with the result
//
// Returns a done channel that closes when processing completes, and an error if validation fails.
func NewClientStreamProcessor[InT, OutT any](ctx context.Context, stream ClientStream[InT, OutT], refsCh <-chan *InT, receiverFn ClientReceiverFn[OutT]) (<-chan struct{}, error) {
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
				// Context cancelled, notify receiver
				_ = receiverFn(nil, ctx.Err())
				return

			case recordRef, ok := <-refsCh:
				// Once the channel is closed, send the data through the stream and exit.
				// Handle any error using the receiver function.
				if !ok {
					resp, err := stream.CloseAndRecv()
					_ = receiverFn(resp, err)

					return
				}

				// Check context again before potentially blocking Send operation
				select {
				case <-ctx.Done():
					_ = receiverFn(nil, ctx.Err())
					return
				default:
				}

				// Send the record reference to the network buffer and handle errors
				if err := stream.Send(recordRef); err != nil {
					// Notify receiver of send error and stop processing
					_ = receiverFn(nil, err)
					return
				}
			}
		}
	}()

	return doneCh, nil
}
