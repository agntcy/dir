// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
)

const (
	// defaultBufferSize is the default buffer size for streaming result channels.
	defaultBufferSize = 10
)

// PushResult represents the result of a push operation.
type PushResult struct {
	RecordRef *corev1.RecordRef
	Error     error
	Index     int // For correlating with input order if needed
}

// PushStream handles streaming push operations in a self-contained manner.
// This follows the generator pattern from "Concurrency in Go" by Katherine Cox-Buday
// where functions take a context, input channel, and configuration, return an output channel,
// and manage their own goroutine lifecycle internally.
//
//nolint:gocognit,cyclop // Streaming functions necessarily have high complexity due to concurrent patterns
func PushStream(ctx context.Context, inStream <-chan *corev1.Record, client storetypes.StoreServiceClient) <-chan PushResult {
	outStream := make(chan PushResult, defaultBufferSize) // Buffer for better performance

	go func() {
		defer close(outStream)

		// Create streaming client once
		stream, err := client.Push(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case outStream <- PushResult{Error: fmt.Errorf("failed to create push stream: %w", err)}:
			}

			return
		}

		var wg sync.WaitGroup

		// Goroutine 1: Send records to server
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer func() {
				if err := stream.CloseSend(); err != nil {
					select {
					case <-ctx.Done():
						return
					case outStream <- PushResult{Error: fmt.Errorf("failed to close send stream: %w", err)}:
					}
				}
			}()

			index := 0

			for record := range inStream {
				select {
				case <-ctx.Done():
					return
				default:
					if err := stream.Send(record); err != nil {
						select {
						case <-ctx.Done():
							return
						case outStream <- PushResult{Error: fmt.Errorf("failed to send record %d: %w", index, err), Index: index}:
						}

						return
					}

					index++
				}
			}
		}()

		// Goroutine 2: Receive responses from server
		wg.Add(1)

		go func() {
			defer wg.Done()

			index := 0

			for {
				recordRef, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				if err != nil {
					select {
					case <-ctx.Done():
						return
					case outStream <- PushResult{Error: fmt.Errorf("failed to receive record ref %d: %w", index, err), Index: index}:
					}

					return
				}

				select {
				case <-ctx.Done():
					return
				case outStream <- PushResult{RecordRef: recordRef, Index: index}:
				}

				index++
			}
		}()

		wg.Wait()
	}()

	return outStream
}
