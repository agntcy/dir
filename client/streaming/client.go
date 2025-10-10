// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)

// ClientReceiverFn is a callback function type for handling responses from the server.
// It takes a pointer to the output type and an error as parameters, and returns an error.
// This function is used to process the server's response after sending data through a gRPC stream.
type ClientReceiverFn[OutT any] func(*OutT, error) error

// ClientStream is a generic interface that defines the methods required for a gRPC client stream.
type ClientStream[InT, OutT any] interface {
	Send(*InT) error
	CloseAndRecv() (*OutT, error)
	grpc.ClientStream
}

// NewClientStreamProcessor creates and starts a goroutine to process incoming record references from a channel
// and send them through a gRPC client stream. It also handles the server's response using a provided receiver function.
// The function takes a context for cancellation, a gRPC client stream, a channel of input references, and a receiver function.
// It returns a channel that signals when processing is done and an error if any validation fails.
func NewClientStreamProcessor[InT, OutT any](ctx context.Context, stream ClientStream[InT, OutT], refsCh <-chan *InT, receiverFn ClientReceiverFn[OutT]) (<-chan struct{}, error) {
	// Validate input parameters
	if refsCh == nil {
		return nil, errors.New("refs channel is nil")
	}

	if receiverFn == nil {
		return nil, errors.New("receiver function is nil")
	}

	// Done channel
	doneCh := make(chan struct{})

	// Process items
	go func() {
		// Close stream and send notification once the goroutine ends
		//nolint:errcheck
		defer stream.CloseSend()
		defer close(doneCh)

		// Process incoming record references
		for {
			select {
			case <-ctx.Done():
				return
			case recordRef, ok := <-refsCh:
				// Once the channel is closed, send the data through the stream and exit.
				// Handle any error using the receiver function.
				if !ok {
					resp, err := stream.CloseAndRecv()
					_ = receiverFn(resp, err)

					return
				}

				// Send the record reference to the network buffer
				_ = stream.Send(recordRef)
			}
		}
	}()

	return doneCh, nil
}
