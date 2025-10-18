// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// ServerStream defines the interface for server streaming (one input → many outputs).
// This pattern is used when sending a single request and receiving multiple responses.
type ServerStream[OutT any] interface {
	Recv() (*OutT, error)
}

// ProcessServerStream handles server streaming pattern (one input → many outputs).
//
// Pattern: Send Request → Recv() → Recv() → Recv() → EOF
//
// This processor is ideal for operations where a single request triggers multiple
// responses from the server, such as:
//   - Streaming search results
//   - Listing resources
//   - Tailing logs
//   - Real-time event streams
//
// The processor:
//   - Continuously receives outputs from the stream
//   - Sends each output to the result channel
//   - Handles EOF gracefully to signal completion
//   - Propagates errors to the error channel
//
// Returns:
//   - result: StreamResult containing result, error, and done channels
//   - error: Immediate error if validation fails
//
// The caller should:
//  1. Range over result channels to process outputs and errors
//  2. Monitor the DoneCh to know when streaming is complete
//  3. Use context cancellation to stop processing early
//
// Example usage:
//
//	stream, err := client.Listen(ctx, req)
//	if err != nil {
//	    return err
//	}
//
//	result, err := streaming.ProcessServerStream(ctx, stream)
//	if err != nil {
//	    return err
//	}
//
//	for {
//	    select {
//	    case resp := <-result.ResCh():
//	        // Process response
//	    case err := <-result.ErrCh():
//	        // Handle error
//	        return err
//	    case <-result.DoneCh():
//	        // All responses received
//	        return nil
//	    case <-ctx.Done():
//	        return ctx.Err()
//	    }
//	}
func ProcessServerStream[OutT any](
	ctx context.Context,
	stream ServerStream[OutT],
) (StreamResult[OutT], error) {
	// Validate inputs
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	if stream == nil {
		return nil, errors.New("stream is nil")
	}

	// Create result channels
	result := newResult[OutT]()

	// Start receiver goroutine
	go func() {
		// Close result once the goroutine ends
		defer result.close()

		// Receive output from the stream
		//
		// Note: stream.Recv() is blocking until a message is available or
		// an error occurs. This provides natural pacing with the server.
		//
		// If the context is cancelled, Recv() will return an error,
		// which terminates this goroutine.
		for {
			output, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				// Normal completion - server closed the stream
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

	return result, nil
}
