// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/config"
)

// ClientPusher is a Pusher implementation that uses the DIR client.
type ClientPusher struct {
	client config.ClientInterface
}

// NewClientPusher creates a new ClientPusher.
func NewClientPusher(client config.ClientInterface) *ClientPusher {
	return &ClientPusher{
		client: client,
	}
}

// Push sends a record to DIR using the client.
func (p *ClientPusher) Push(ctx context.Context, inputCh <-chan *corev1.Record) (<-chan *corev1.RecordRef, <-chan error) {
	refCh := make(chan *corev1.RecordRef)
	errCh := make(chan error)

	go func() {
		defer close(refCh)
		defer close(errCh)

		// Push records to DIR using the streaming API.
		streamResult, err := p.client.PushStream(ctx, inputCh)
		if err != nil {
			errCh <- err

			return
		}

		// Process stream results
		for {
			select {
			case ref, ok := <-streamResult.ResCh():
				if !ok {
					continue
				}

				select {
				case refCh <- ref:
				case <-ctx.Done():
					return
				}
			case err, ok := <-streamResult.ErrCh():
				if !ok {
					continue
				}

				select {
				case errCh <- err:
				case <-ctx.Done():
					return
				}
			case <-streamResult.DoneCh():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return refCh, errCh
}
