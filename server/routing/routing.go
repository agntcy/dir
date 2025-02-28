// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/types"
)

type routing struct {
	ds types.Datastore
}

func New(opts types.APIOptions) (types.RoutingAPI, error) {
	return &routing{
		ds: opts.Datastore(),
	}, nil
}

func (r *routing) List(ctx context.Context, key types.Key, filters string, readerFn func(types.Key, coretypes.ObjectRef) error) error {
	panic("unimplemented")
}

func (r *routing) Lookup(context.Context, types.Key) (*coretypes.ObjectRef, error) {
	panic("unimplemented")
}

func (r *routing) Publish(context.Context, *coretypes.ObjectRef) error {
	panic("unimplemented")
}

func (r *routing) Resolve(context.Context, types.Key) (<-chan *types.Peer, error) {
	panic("unimplemented")
}
