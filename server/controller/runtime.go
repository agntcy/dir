// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/agntcy/dir/server/types"
)

type runtimeCtlr struct {
	runtimev1.UnimplementedDiscoveryServiceServer
	api types.DiscoveryAPI
}

// NewRuntimeController creates a new runtime controller.
func NewRuntimeController(api types.DiscoveryAPI) runtimev1.DiscoveryServiceServer {
	return &runtimeCtlr{
		UnimplementedDiscoveryServiceServer: runtimev1.UnimplementedDiscoveryServiceServer{},
		api:                                 api,
	}
}

func (c *runtimeCtlr) ListProcesses(req *runtimev1.ListProcessesRequest, stream runtimev1.DiscoveryService_ListProcessesServer) error {
	return c.api.ListProcesses(stream.Context(), req, stream.Send)
}
