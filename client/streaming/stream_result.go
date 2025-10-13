// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

// StreamResult encapsulates the channels for receiving streaming results,
// errors, and completion signals. It provides a structured way to handle
// streaming responses.
//
// NOTES:
//   - For ClientStream, the ResCh can receive a single result before the DoneCh is closed.
//   - For BidiStream, the ResCh can receive multiple results until the DoneCh is closed.
type StreamResult[OutT any] interface {
	ResCh() <-chan *OutT
	ErrCh() <-chan error
	DoneCh() <-chan struct{}
}

// result is a concrete implementation of StreamResult.
type result[OutT any] struct {
	resCh  chan *OutT
	errCh  chan error
	doneCh chan struct{}
}

func newResult[OutT any]() *result[OutT] {
	return &result[OutT]{
		resCh:  make(chan *OutT),
		errCh:  make(chan error),
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

// close closes only the control channel doneCh to signal completion.
func (r *result[OutT]) close() {
	close(r.doneCh)
}
