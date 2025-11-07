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

// BidiStream defines the interface for bidirectional streaming.
// This pattern allows sending and receiving messages independently.
type BidiStream[InT, OutT any] interface {
	Send(*InT) error
	Recv() (*OutT, error)
	CloseSend() error
}

// ProcessBidiStream handles concurrent bidirectional streaming.
//
// Pattern: Sender || Receiver (parallel goroutines)
//
// This processor implements true bidirectional streaming with concurrent send and receive operations.
// It spawns two independent goroutines:
//   - Sender: Continuously sends inputs from the input channel
//   - Receiver: Continuously receives outputs and sends them to the output channel
//
// This pattern maximizes throughput by:
//   - Eliminating round-trip latency between requests
//   - Allowing the server to batch/buffer/pipeline operations
//   - Fully utilizing network bandwidth
//   - Enabling concurrent processing on both client and server
//
// This is useful when:
//   - High performance and throughput are needed
//   - Processing large batches of data
//   - Server can process requests in parallel or batches
//   - Responses can arrive in any order or timing
//   - Network latency is significant
//
// Returns:
//   - result: StreamResult containing result, error, and done channels
//   - error: Immediate error if validation fails
//
// The caller should:
//  1. Range over result channels to process outputs and errors
//  2. Check if the processing is done
//  3. Use context cancellation to stop processing early
func ProcessBidiStream[InT, OutT any](
	ctx context.Context,
	stream BidiStream[InT, OutT],
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

	// Start goroutines
	go func() {
		// Close result once the goroutine ends
		defer result.close()

		// WaitGroup to coordinate send/receive goroutines
		var wg sync.WaitGroup

		// Goroutine [Sender]: send all inputs from inputCh to the network,
		// then close the send side of the stream.
		// On error, stop and report the error.
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Close the send side when done sending inputs
			//nolint:errcheck
			defer stream.CloseSend()

			// Drain inputCh on early exit to prevent upstream producers from blocking
			defer func() {
				for range inputCh {
					// Drain remaining inputs
				}
			}()

			// Send input to the stream
			//
			// Note: stream.Send() is blocking if the internal buffer is full.
			// This provides backpressure to the sender goroutine
			// which in turn provides backpressure to the input channel
			// and upstream producers.
			//
			// If the context is cancelled, Send() will return an error,
			// which terminates this goroutine.
			for input := range inputCh {
				if err := stream.Send(input); err != nil {
					result.errCh <- fmt.Errorf("failed to send: %w", err)

					return
				}
			}
		}()

		// Goroutine [Receiver]: receive all responses from API and send them to outputCh.
		// On error, stop and report the error.
		wg.Add(1)

		go func() {
			defer wg.Done()

			// Receive output from the stream
			//
			// Note: stream.Recv() is blocking until a message is available or
			// an error occurs. This provides natural pacing with the server.
			//
			// If the context is cancelled, Send() will return an error,
			// which terminates this goroutine.
			for {
				output, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					return
				}

				if err != nil {
					result.errCh <- fmt.Errorf("failed to receive: %w", err)

					return
				}

				// Send output to the output channel
				result.resCh <- output
			}
		}()

		// Wait for all goroutines to complete
		wg.Wait()
	}()

	return result, nil
}
