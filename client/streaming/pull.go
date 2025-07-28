// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"fmt"
	"io"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
)

// PullResult represents the result of a pull operation
type PullResult struct {
	Record *corev1.Record
	Error  error
	Index  int // For correlating with input order if needed
}

// PullStream handles streaming pull operations in a self-contained manner.
// This follows the generator pattern from "Concurrency in Go" by Katherine Cox-Buday
// where functions take a context, input channel, and configuration, return an output channel,
// and manage their own goroutine lifecycle internally.
func PullStream(ctx context.Context, inStream <-chan *corev1.RecordRef, client storetypes.StoreServiceClient) <-chan PullResult {
	outStream := make(chan PullResult)

	go func() {
		defer close(outStream)

		// Create gRPC stream once
		stream, err := client.Pull(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case outStream <- PullResult{Error: fmt.Errorf("failed to create pull stream: %w", err)}:
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
					case outStream <- PullResult{Error: fmt.Errorf("failed to close send stream: %w", err)}:
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
						case outStream <- PullResult{Error: fmt.Errorf("failed to send record ref %d: %w", index, err), Index: index}:
						}
						return
					}
					index++
				}
			}
		}()

		// Goroutine 2: Receive records from server
		wg.Add(1)
		go func() {
			defer wg.Done()
			index := 0
			for {
				record, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case outStream <- PullResult{Error: fmt.Errorf("failed to receive record %d: %w", index, err), Index: index}:
					}
					return
				}

				select {
				case <-ctx.Done():
					return
				case outStream <- PullResult{Record: record, Index: index}:
				}
				index++
			}
		}()

		wg.Wait()
	}()

	return outStream
}
