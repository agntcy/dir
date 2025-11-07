// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"
	"fmt"
)

// ClientStream defines the interface for client streaming (many inputs → one output).
// This pattern is used when sending multiple requests and receiving a single response.
type ClientStream[InT, OutT any] interface {
	Send(*InT) error
	CloseAndRecv() (*OutT, error)
	CloseSend() error
}

// ProcessClientStream handles client streaming pattern (many inputs → one output).
//
// Pattern: Send → Send → Send → CloseAndRecv()
//
// This processor is ideal for operations where multiple requests are sent to the server,
// and a single final response is received after all requests have been processed.
//
// The processor:
//   - Sends all inputs from the channel to the stream
//   - Closes the send side when input channel closes
//   - Receives the final response via CloseAndRecv()
//
// Returns:
//   - result: StreamResult containing result, error, and done channels
//   - error: Immediate error if validation fails
//
// The caller should:
//  1. Range over result channels to process outputs and errors
//  2. Check if the processing is done
//  3. Use context cancellation to stop processing early
func ProcessClientStream[InT, OutT any](
	ctx context.Context,
	stream ClientStream[InT, OutT],
	inputCh <-chan *InT,
) (StreamResult[OutT], error) {
	// Validate inputs
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	if stream == nil {
		return nil, errors.New("stream is nil")
	}

	if inputCh == nil {
		return nil, errors.New("input channel is nil")
	}

	// Create result channels
	result := newResult[OutT]()

	// Process items
	go func() {
		// Close result once the goroutine ends
		defer result.close()

		// Close the send side when done sending inputs
		//nolint:errcheck
		defer stream.CloseSend()

		// Drain inputCh on early exit to prevent upstream producers from blocking
		defer func() {
			for range inputCh {
				// Drain remaining inputs
			}
		}()

		// Process all incoming inputs
		for input := range inputCh {
			// Send the input to the network buffer and handle errors
			if err := stream.Send(input); err != nil {
				result.errCh <- fmt.Errorf("failed to send: %w", err)

				return
			}
		}

		// Once the channel is closed, send the data through the stream and exit.
		// Handle any errors using the error handler function.
		resp, err := stream.CloseAndRecv()
		if err != nil {
			result.errCh <- fmt.Errorf("failed to receive final response: %w", err)

			return
		}

		// Send the final response to the output channel
		result.resCh <- resp
	}()

	return result, nil
}
