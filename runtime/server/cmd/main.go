// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/server/config"
	"github.com/agntcy/dir/runtime/server/database"
	grpcserver "github.com/agntcy/dir/runtime/server/grpc"
	"github.com/agntcy/dir/runtime/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var logger = utils.NewLogger("process", "server")

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", "error", err)
	}

	logger.Info("============================================================")
	logger.Info("Discovery Server")
	logger.Info("configuration loaded",
		"server", cfg.Addr(),
		"storage", cfg.Store.Type,
	)
	logger.Info("============================================================")

	// Initialize database
	db, err := database.NewDatabase(cfg.Store)
	if err != nil {
		logger.Fatal("Failed to initialize database", "error", err)
	}
	defer db.Close()

	// Create gRPC server
	grpcServer := grpc.NewServer()
	discoveryServer := grpcserver.NewDiscovery(db)
	runtimev1.RegisterDiscoveryServiceServer(grpcServer, discoveryServer)

	// Enable reflection for debugging tools like grpcurl
	reflection.Register(grpcServer)

	// Start listener
	//nolint:noctx
	listener, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		logger.Fatal("Failed to listen", "address", cfg.Addr(), "error", err)
	}

	// Start server in goroutine
	go func() {
		logger.Info("gRPC server listening", "address", cfg.Addr())

		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatal("Server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down server...")
	grpcServer.GracefulStop()
}
