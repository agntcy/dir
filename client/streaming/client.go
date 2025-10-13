// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"
	"fmt"
)

// 2. Use context cancellation to stop processing early.
func NewClientStreamProcessor[InT, OutT any](
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
