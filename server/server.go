// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Portshift/go-utils/healthz"
	apitypes "github.com/agntcy/dir/api/store/v1alpha1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	listenAddress string
	healthzServer *healthz.Server
	grpcServer    *grpc.Server
}

func New(config *config.Config) (*Server, error) {
	// Create provider
	provider, err := store.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Create server
	s := &Server{
		listenAddress: config.ListenAddress,
		healthzServer: healthz.NewHealthServer(config.HealthCheckAddress),
		grpcServer:    grpc.NewServer(),
	}

	// Register APIs
	// TODO: add routing
	apitypes.RegisterStoreServiceServer(s.grpcServer, controller.NewStoreController(provider))

	// Register server
	reflection.Register(s.grpcServer)

	return s, nil
}

func (s Server) Start(errCh chan<- error) {
	// Create a listener on TCP port
	listen, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		errCh <- fmt.Errorf("failed to listen on %s: %w", s.listenAddress, err)

		return
	}

	// Serve gRPC server
	go func() {
		// Start health check server
		s.healthzServer.Start()
		s.healthzServer.SetIsReady(true)
		defer s.healthzServer.SetIsReady(false)

		if err := s.grpcServer.Serve(listen); err != nil {
			errCh <- fmt.Errorf("failed to start server: %w", err)

			return
		}
	}()
}

func (s Server) Stop() {
	s.grpcServer.GracefulStop()
}

func Run(ctx context.Context, cfg *config.Config) error {
	errCh := make(chan error)

	// Start server
	server, err := New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	server.Start(errCh)
	defer server.Stop()
	log.Printf("server started on %s", cfg.ListenAddress)

	// Wait for deactivation
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("stopping server due to context cancellation: %w", ctx.Err())
	case sig := <-sigCh:
		return fmt.Errorf("stopping server due to signal: %v", sig)
	case err := <-errCh:
		return fmt.Errorf("stopping server due to error: %w", err)
	}
}
