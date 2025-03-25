// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
)

type route struct {
	local  *routeLocal
	remote *routeRemote
}

func New(ctx context.Context, store types.StoreAPI, opts types.APIOptions) (types.RoutingAPI, error) {
	local := newLocal(store, opts.Datastore())

	remote, err := newRemote(ctx, store, opts)
	if err != nil {
		return nil, err
	}

	return &route{
		local:  local,
		remote: remote,
	}, nil
}

func (r *route) Publish(ctx context.Context, obj *coretypes.Object, isLocal bool) error {
	if isLocal {
		return r.local.Publish(ctx, obj, isLocal)
	}

	return r.remote.Publish(ctx, obj, isLocal)
}

func (r *route) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	if req.GetLocal() {
		return r.local.List(ctx, req)
	}

	return r.remote.List(ctx, req)
}
