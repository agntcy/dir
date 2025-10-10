// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)

// ServerReceiverFn is a callback function type for handling responses from the server.
// It takes a pointer to the output type and an error as parameters, and returns an error.
// This function is used to process the server's response after sending data through a gRPC stream.
type ServerReceiverFn[InT, OutT any] func(*InT, *OutT, error) error

// ServerStream is a generic interface that defines the methods required for a gRPC server stream.
type ServerStream[InT, OutT any] interface {
	Send(*InT) error
	Recv() (*OutT, error)
	grpc.ClientStream
}

// NewServerStreamProcessor creates and starts a goroutine to process incoming record references from a channel
// and send them through a gRPC server stream. It also handles the server's response using a provided receiver function.
// The function takes a context for cancellation, a gRPC server stream, a channel of input references, and a receiver function.
// It returns a channel that signals when processing is done and an error if any validation fails.
func NewServerStreamProcessor[InT, OutT any](ctx context.Context, stream ServerStream[InT, OutT], refsCh <-chan *InT, receiverFn ServerReceiverFn[InT, OutT]) (<-chan struct{}, error) {
	// Validate input parameters
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	if stream == nil {
		return nil, errors.New("stream is nil")
	}

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
				// Exit if channel is closed
				if !ok {
					return
				}

				// Check context again before potentially blocking Send operation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Send the record reference to the network buffer
				// Handle any error using the receiver function
				err := stream.Send(recordRef)
				if err != nil {
					if recvErr := receiverFn(recordRef, nil, err); recvErr != nil {
						return
					}

					continue
				}

				// Receive the response from the network buffer
				// Handle the response and any error using the receiver function
				response, err := stream.Recv()
				if recvErr := receiverFn(recordRef, response, err); recvErr != nil {
					return
				}
			}
		}
	}()

	return doneCh, nil
}
