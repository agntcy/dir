// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

// NewBidirectionalStreamProcessor handles concurrent bidirectional streaming.
//
// Pattern: Sender || Receiver (parallel goroutines)
//
// This processor implements true bidirectional streaming with concurrent send and receive operations.
// It spawns two independent goroutines:
// - Sender: Continuously sends inputs from the input channel
// - Receiver: Continuously receives outputs and sends them to the output channel
//
// This pattern maximizes throughput by:
// - Eliminating round-trip latency between requests
// - Allowing the server to batch/buffer/pipeline operations
// - Fully utilizing network bandwidth
// - Enabling concurrent processing on both client and server
//
// This is useful when:
// - High performance and throughput are needed
// - Processing large batches of data
// - Server can process requests in parallel or batches
// - Responses can arrive in any order or timing
// - Network latency is significant
//
// Returns:
// - outputCh: Channel of output items as they arrive from the server
// - errCh: Buffered error channel (contains at most one error)
// - error: Immediate error if validation fails
//
// The caller should:
// 1. Range over outputCh to process results
// 2. Check errCh after outputCh closes to detect errors
// 3. Use context cancellation to stop processing early
func NewBidirectionalStreamProcessor[InT, OutT any](
	ctx context.Context,
	stream BidirectionalStream[InT, OutT],
	inputCh <-chan *InT,
	validateOutput OutputValidatorFn[OutT],
) (<-chan *OutT, <-chan error, error) {
	outputCh := make(chan *OutT)
	errCh := make(chan error, 1)

	// Validate inputs
	if ctx == nil {
		errCh <- errors.New("context is nil")
		close(outputCh)
		close(errCh)
		return outputCh, errCh, errors.New("context is nil")
	}

	if stream == nil {
		errCh <- errors.New("stream is nil")
		close(outputCh)
		close(errCh)
		return outputCh, errCh, errors.New("stream is nil")
	}

	if inputCh == nil {
		errCh <- errors.New("input channel is nil")
		close(outputCh)
		close(errCh)
		return outputCh, errCh, errors.New("input channel is nil")
	}

	go func() {
		defer close(outputCh)
		defer close(errCh)

		// WaitGroup to coordinate sender and receiver goroutines
		var wg sync.WaitGroup

		// Error channel for internal goroutine communication
		internalErrCh := make(chan error, 2)

		// Sender goroutine: send all inputs
		wg.Add(1)
		go func() {
			defer wg.Done()

			for input := range inputCh {
				// Check context before sending to avoid unnecessary work
				select {
				case <-ctx.Done():
					internalErrCh <- ctx.Err()
					return
				default:
				}

				if err := stream.Send(input); err != nil {
					internalErrCh <- fmt.Errorf("failed to send: %w", err)
					return
				}
			}

			// Close the send side when done sending
			if err := stream.CloseSend(); err != nil {
				internalErrCh <- fmt.Errorf("failed to close send stream: %w", err)
			}
		}()

		// Receiver goroutine: receive all outputs
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				output, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					return
				}

				if err != nil {
					internalErrCh <- fmt.Errorf("failed to receive: %w", err)
					return
				}

				// Optional validation
				if validateOutput != nil {
					if err := validateOutput(output); err != nil {
						internalErrCh <- fmt.Errorf("validation failed: %w", err)
						return
					}
				}

				select {
				case outputCh <- output:
				case <-ctx.Done():
					internalErrCh <- ctx.Err()
					return
				}
			}
		}()

		// Wait for both goroutines to complete
		wg.Wait()

		// Collect the first error if any occurred
		select {
		case err := <-internalErrCh:
			errCh <- err
			// Drain any remaining error to avoid goroutine leak
			select {
			case <-internalErrCh:
			default:
			}
		default:
		}
	}()

	return outputCh, errCh, nil
}
