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
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/routing"
	"github.com/agntcy/dir/server/store"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var _ types.API = &Server{}

type Server struct {
	options       types.APIOptions
	store         types.StoreAPI
	routing       types.RoutingAPI
	healthzServer *healthz.Server
	grpcServer    *grpc.Server
}

func Run(ctx context.Context, cfg *config.Config) error {
	errCh := make(chan error)

	// Start server
	server, err := New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	server.Start(ctx, errCh)
	defer server.Stop()
	log.Printf("server started on %s", cfg.ListenAddress)

	// Wait for deactivation
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck
	case sig := <-sigCh:
		return fmt.Errorf("stopping server due to signal: %v", sig)
	case err := <-errCh:
		return fmt.Errorf("stopping server due to error: %w", err)
	}
}

func New(cfg *config.Config) (*Server, error) {
	// Create local datastore
	dstore, err := datastore.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	// Load API options
	options := &options{
		config:    cfg,
		datastore: dstore,
	}

	// Create APIs
	storeAPI, err := store.New(options) //nolint:staticcheck
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	routingAPI, err := routing.New(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create routing: %w", err)
	}

	// Create server
	server := &Server{
		options:       options,
		store:         storeAPI,
		routing:       routingAPI,
		healthzServer: healthz.NewHealthServer(cfg.HealthCheckAddress),
		grpcServer:    grpc.NewServer(),
	}

	// Register APIs
	storetypes.RegisterStoreServiceServer(server.grpcServer, controller.NewStoreController(storeAPI))
	routingtypes.RegisterRoutingServiceServer(server.grpcServer, controller.NewRoutingController(routingAPI))

	// Register server
	reflection.Register(server.grpcServer)

	return server, nil
}

func (s Server) Options() types.APIOptions { return s.options }

func (s Server) Store() types.StoreAPI { return s.store }

func (s Server) Routing() types.RoutingAPI { return s.routing }

func (s Server) Start(ctx context.Context, errCh chan<- error) {
	// Bootstrap
	if err := s.bootstrap(ctx); err != nil {
		errCh <- fmt.Errorf("failed to bootstrap server: %w", err)

		return
	}

	// Create a listener on TCP port
	listen, err := net.Listen("tcp", s.Options().Config().ListenAddress)
	if err != nil {
		errCh <- fmt.Errorf("failed to listen on %s: %w", s.Options().Config().ListenAddress, err)

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

func (s Server) bootstrap(_ context.Context) error {
	// TODO: bootstrap routing and storage data by listing from storage
	// TODO: also update cache datastore
	return nil
}

type options struct {
	config    *config.Config
	datastore types.Datastore
}

func (o options) Config() *config.Config { return o.config }

func (o options) Datastore() types.Datastore { return o.datastore }
