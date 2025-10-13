// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

import "google.golang.org/grpc"

// BidiStream defines the interface for bidirectional streaming.
// This pattern allows sending and receiving messages independently.
type BidiStream[InT, OutT any] interface {
	Send(*InT) error
	Recv() (*OutT, error)
	CloseSend() error
	grpc.ClientStream
}

// ClientStream defines the interface for client streaming (many inputs → one output).
// This pattern is used when sending multiple requests and receiving a single response.
type ClientStream[InT, OutT any] interface {
	Send(*InT) error
	CloseAndRecv() (*OutT, error)
	grpc.ClientStream
}

type StreamResult[OutT any] interface {
	ResCh() <-chan *OutT
	ErrCh() <-chan error
	DoneCh() <-chan struct{}
}

type result[OutT any] struct {
	resCh  chan *OutT
	errCh  chan error
	doneCh chan struct{}
}

func newResult[OutT any]() *result[OutT] {
	return &result[OutT]{
		resCh:  make(chan *OutT),
		errCh:  make(chan error, 1),
		doneCh: make(chan struct{}),
	}
}

func (r *result[OutT]) ResCh() <-chan *OutT {
	return r.resCh
}

func (r *result[OutT]) ErrCh() <-chan error {
	return r.errCh
}

func (r *result[OutT]) DoneCh() <-chan struct{} {
	return r.doneCh
}

func (r *result[OutT]) close() {
	close(r.resCh)
	close(r.errCh)
	close(r.doneCh)
}
