// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:nlreturn
package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/agntcy/dir/runtime/server"
	"github.com/agntcy/dir/runtime/server/config"
	"github.com/agntcy/dir/runtime/store"
	"github.com/agntcy/dir/runtime/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var logger = utils.NewLogger("runtime", "server")

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		return
	}

	logger.Info("============================================================")
	logger.Info("Discovery Server")
	logger.Info("configuration loaded",
		"server", cfg.Addr(),
		"storage", cfg.Store.Type,
	)
	logger.Info("============================================================")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create store
	store, err := store.New(cfg.Store)
	if err != nil {
		logger.Error("Failed to create store", "error", err)
		return
	}
	defer store.Close()

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register runtime server
	err = server.Register(grpcServer, store)
	if err != nil {
		logger.Error("failed to initialize runtime server internals", "error", err)
		return
	}

	// Register reflection service on gRPC server
	reflection.Register(grpcServer)

	//nolint:noctx
	listener, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		logger.Error("failed to listen", "address", cfg.Addr(), "error", err)
		return
	}

	serverErrCh := make(chan error, 1)

	// Start discovery service in a separate goroutine
	go func() {
		logger.Info("gRPC server listening", "address", cfg.Addr())

		if err := grpcServer.Serve(listener); err != nil {
			serverErrCh <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("received shutdown signal")
		grpcServer.GracefulStop()
	case err := <-serverErrCh:
		logger.Error("server failed", "error", err)
	}
}
