// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/server/database"
	grpcserver "github.com/agntcy/dir/runtime/server/grpc"
	storetypes "github.com/agntcy/dir/runtime/store/types"
	"google.golang.org/grpc"
)

// Register registers runtime services on the provided gRPC registrar.
func Register(server *grpc.Server, store storetypes.StoreReader) error {
	// Create database
	db, err := database.NewDatabase(store)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Register discovery service
	discoveryServer := grpcserver.NewDiscovery(db)
	runtimev1.RegisterDiscoveryServiceServer(server, discoveryServer)

	return nil
}
