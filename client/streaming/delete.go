// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
)

// DeleteResult represents the result of a delete operation
type DeleteResult struct {
	Error error
	Index int // For correlating with input order if needed
}

// DeleteStream handles streaming delete operations in a self-contained manner.
// This follows the generator pattern from "Concurrency in Go" by Katherine Cox-Buday
// where functions take a context, input channel, and configuration, return an output channel,
// and manage their own goroutine lifecycle internally.
func DeleteStream(ctx context.Context, inStream <-chan *corev1.RecordRef, client storetypes.StoreServiceClient) <-chan DeleteResult {
	outStream := make(chan DeleteResult)

	go func() {
		defer close(outStream)

		// Create gRPC stream once
		stream, err := client.Delete(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case outStream <- DeleteResult{Error: fmt.Errorf("failed to create delete stream: %w", err)}:
			}
			return
		}

		// Send all record refs and emit results
		index := 0
		for recordRef := range inStream {
			select {
			case <-ctx.Done():
				return
			default:
				if err := stream.Send(recordRef); err != nil {
					select {
					case <-ctx.Done():
						return
					case outStream <- DeleteResult{Error: fmt.Errorf("failed to send record ref %d: %w", index, err), Index: index}:
					}
					return
				}

				// Send successful - emit success result
				select {
				case <-ctx.Done():
					return
				case outStream <- DeleteResult{Index: index}:
				}
				index++
			}
		}

		// Close the send stream and wait for server confirmation
		_, err = stream.CloseAndRecv()
		if err != nil && err != io.EOF {
			select {
			case <-ctx.Done():
				return
			case outStream <- DeleteResult{Error: fmt.Errorf("failed to close delete stream: %w", err)}:
			}
			return
		}
	}()

	return outStream
}
