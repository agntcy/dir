// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore/query"
)

type routing struct {
	ds types.Datastore
}

func New(opts types.APIOptions) (types.RoutingAPI, error) {
	return &routing{
		ds: opts.Datastore(),
	}, nil
}

func (r *routing) Publish(context.Context, *coretypes.ObjectRef, *coretypes.Agent) error {
	panic("unimplemented")
}

func (r *routing) List(context.Context, query.Query) ([]*coretypes.ObjectRef, error) {
	panic("unimplemented")
}
