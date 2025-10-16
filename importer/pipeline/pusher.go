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
func (p *ClientPusher) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	//nolint:wrapcheck
	return p.client.Push(ctx, record)
}
