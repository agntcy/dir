// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc/status"
)

type route struct {
	local  *routeLocal
	remote *routeRemote
}

func New(ctx context.Context, store types.StoreAPI, opts types.APIOptions) (types.RoutingAPI, error) {
	// Create main router
	mainRounter := &route{}

	// Create routing datastore
	var dsOpts []datastore.Option
	if dstoreDir := opts.Config().Routing.DatastoreDir; dstoreDir != "" {
		dsOpts = append(dsOpts, datastore.WithFsProvider(dstoreDir))
	}

	dstore, err := datastore.New(dsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create routing datastore: %w", err)
	}

	// Create local router
	mainRounter.local = newLocal(store, dstore)

	// Create remote router
	mainRounter.remote, err = newRemote(ctx, mainRounter, store, dstore, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote routing: %w", err)
	}

	return mainRounter, nil
}

func (r *route) Publish(ctx context.Context, object *coretypes.Object, network bool) error {
	// always publish data locally for archival/querying
	err := r.local.Publish(ctx, object, network)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to publish locally: %s", st.Message())
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

func (r *route) Unpublish(ctx context.Context, object *coretypes.Object, _ bool) error {
	err := r.local.Unpublish(ctx, object)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to unpublish locally: %s", st.Message())
	}

	// no need to explicitly handle unpublishing from the network
	// TODO clarify if network sync trigger is needed here
	return nil
}
