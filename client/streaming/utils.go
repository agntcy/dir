// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import "context"

// SliceToChan converts a slice of items into a channel that emits each item.
// It respects the provided context for cancellation.
func SliceToChan[T any](ctx context.Context, items []T) <-chan T {
	outCh := make(chan T, len(items))

	go func() {
		defer close(outCh)

		for _, item := range items {
			select {
			case outCh <- item:
			case <-ctx.Done():
				return
			}
		}
	}()

	return outCh
}
