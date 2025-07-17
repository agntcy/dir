// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
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

func (r *route) Publish(ctx context.Context, object *coretypes.Object) error {
	// always publish data locally for archival/querying
	err := r.local.Publish(ctx, object)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to publish locally: %s", st.Message())
	}

	err = r.remote.Publish(ctx, object)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to publish to the network: %s", st.Message())
	}

	return nil
}

func (r *route) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.LegacyListResponse_Item, error) {
	// Use remote routing when:
	// 1. Looking for a specific record (digest) - to find providers across the network
	// 2. MaxHops is set - indicates network traversal
	if req.GetLegacyListRequest().GetRecord() != nil || req.GetLegacyListRequest().GetMaxHops() > 0 {
		return r.remote.List(ctx, req)
	}

	// Otherwise use local routing
	return r.local.List(ctx, req)
}

func (r *route) Unpublish(ctx context.Context, object *coretypes.Object) error {
	err := r.local.Unpublish(ctx, object)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to unpublish locally: %s", st.Message())
	}

	// no need to explicitly handle unpublishing from the network
	// TODO clarify if network sync trigger is needed here
	return nil
}
