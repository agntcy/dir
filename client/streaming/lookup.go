// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Streaming functions intentionally follow the same pattern for consistency
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

// LookupResult represents the result of a lookup operation.
type LookupResult struct {
	RecordMeta *corev1.RecordMeta
	Error      error
	Index      int // For correlating with input order if needed
}

// LookupStream handles streaming lookup operations in a self-contained manner.
// This follows the generator pattern from "Concurrency in Go" by Katherine Cox-Buday
// where functions take a context, input channel, and configuration, return an output channel,
// and manage their own goroutine lifecycle internally.
//
//nolint:gocognit,cyclop // Streaming functions necessarily have high complexity due to concurrent patterns
func LookupStream(ctx context.Context, inStream <-chan *corev1.RecordRef, client storetypes.StoreServiceClient) <-chan LookupResult {
	outStream := make(chan LookupResult)

	go func() {
		defer close(outStream)

		// Create gRPC stream once
		stream, err := client.Lookup(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case outStream <- LookupResult{Error: fmt.Errorf("failed to create lookup stream: %w", err)}:
			}

			return
		}

		var wg sync.WaitGroup

		// Goroutine 1: Send record refs to server
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer func() {
				if err := stream.CloseSend(); err != nil {
					select {
					case <-ctx.Done():
						return
					case outStream <- LookupResult{Error: fmt.Errorf("failed to close send stream: %w", err)}:
					}
				}
			}()

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
						case outStream <- LookupResult{Error: fmt.Errorf("failed to send record ref %d: %w", index, err), Index: index}:
						}

						return
					}

					index++
				}
			}
		}()

		// Goroutine 2: Receive record metadata from server
		wg.Add(1)

		go func() {
			defer wg.Done()

			index := 0

			for {
				recordMeta, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				if err != nil {
					select {
					case <-ctx.Done():
						return
					case outStream <- LookupResult{Error: fmt.Errorf("failed to receive record meta %d: %w", index, err), Index: index}:
					}

					return
				}

				select {
				case <-ctx.Done():
					return
				case outStream <- LookupResult{RecordMeta: recordMeta, Index: index}:
				}

				index++
			}
		}()

		wg.Wait()
	}()

	return outStream
}
