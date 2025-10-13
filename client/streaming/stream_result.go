// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package streaming

// StreamResult encapsulates the channels for receiving streaming results,
// errors, and completion signals. It provides a structured way to handle
// streaming responses.
//
// Callers that handle StreamResult should:
//  1. Range over the ResCh to process incoming results.
//  2. Range over the ErrCh to handle errors as they occur.
//  3. Monitor the DoneCh to know when processing is complete.
//
// If the caller does not subscribe to these channels, the processing
// goroutines will block until the channels are read or the context is cancelled.
//
// Example usage:
//
//	for {
//	    select {
//	    case res := <-result.ResCh():
//	        // Process result
//	    case err := <-result.ErrCh():
//	        // Handle error
//	    case <-result.DoneCh():
//	        // Processing is done
//	        // Exit loop
//	        return
//	    }
//	}
type StreamResult[OutT any] interface {
	// ResCh returns a read-only channel for receiving results of type *OutT.
	// More than one result can be sent before the DoneCh is closed.
	//
	// NOTES:
	//   - For ClientStream, the ResCh can receive a single result before the DoneCh is closed.
	//   - For BidiStream, the ResCh can receive multiple results until the DoneCh is closed.
	ResCh() <-chan *OutT

	// ErrCh returns a read-only channel for receiving errors encountered during processing.
	// Errors are sent to this channel as they occur.
	// More than one error can be sent before the DoneCh is closed.
	ErrCh() <-chan error

	// DoneCh returns a read-only channel that is closed when processing is complete.
	// It is used to signal that no more results or errors will be sent.
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
