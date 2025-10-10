// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import "google.golang.org/grpc"

// ClientStream defines the interface for client streaming (many inputs → one output).
// This pattern is used when sending multiple requests and receiving a single response.
type ClientStream[InT, OutT any] interface {
	Send(*InT) error
	CloseAndRecv() (*OutT, error)
	grpc.ClientStream
}

// BidirectionalStream defines the interface for bidirectional streaming.
// This pattern allows sending and receiving messages independently.
type BidirectionalStream[InT, OutT any] interface {
	Send(*InT) error
	Recv() (*OutT, error)
	CloseSend() error
	grpc.ClientStream
}

// ClientReceiverFn is a callback function for handling the final response in client streaming.
// It receives the output and any error that occurred.
type ClientReceiverFn[OutT any] func(*OutT, error) error

// SequentialReceiverFn is a callback function for handling request-response pairs in sequential streaming.
// It receives the input that was sent, the corresponding output, and any error.
type SequentialReceiverFn[InT, OutT any] func(*InT, *OutT, error) error

// OutputValidatorFn is an optional function for validating outputs in bidirectional streaming.
// It receives an output and returns an error if validation fails.
type OutputValidatorFn[OutT any] func(*OutT) error
