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
	mainRounter := &route{
		local: newLocal(store, opts.Datastore()),
	}

	remote, err := newRemote(ctx, mainRounter, store, opts)
	if err != nil {
		return nil, err
	}

	mainRounter.remote = remote

	return mainRounter, nil
}

func (r *route) Publish(ctx context.Context, object *coretypes.Object, network bool) error {
	// always publish data locally for archival/querying
	err := r.local.Publish(ctx, object, network)
	if err != nil {
		return err
	}

	// publish to the network if requested
	if network {
		return r.remote.Publish(ctx, object, network)
	}

	return nil
}

func (r *route) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	if !req.GetNetwork() {
		return r.local.List(ctx, req)
	}

	return r.remote.List(ctx, req)
}
