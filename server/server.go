// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Portshift/go-utils/healthz"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
	searchtypes "github.com/agntcy/dir/api/search/v1alpha2"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/agntcy/dir/api/version"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/database"
	"github.com/agntcy/dir/server/routing"
	"github.com/agntcy/dir/server/store"
	"github.com/agntcy/dir/server/sync"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	_      types.API = &Server{}
	logger           = logging.Logger("server")
)

type Server struct {
	options       types.APIOptions
	store         types.StoreAPI
	routing       types.RoutingAPI
	database      types.DatabaseAPI
	syncService   *sync.Service
	healthzServer *healthz.Server
	grpcServer    *grpc.Server
}

func Run(ctx context.Context, cfg *config.Config) error {
	errCh := make(chan error)

	server, err := New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server
	if err := server.start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer server.Close()

	// Wait for deactivation
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-ctx.Done():
		return fmt.Errorf("stopping server due to context cancellation: %w", ctx.Err())
	case sig := <-sigCh:
		return fmt.Errorf("stopping server due to signal: %v", sig)
	case err := <-errCh:
		return fmt.Errorf("stopping server due to error: %w", err)
	}
}

func New(ctx context.Context, cfg *config.Config) (*Server, error) {
	logger.Debug("Creating server with config", "config", cfg, "version", version.String())

	// Load API options
	options := types.NewOptions(cfg)

	// Create APIs
	storeAPI, err := store.New(options) //nolint:staticcheck
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	routingAPI, err := routing.New(ctx, storeAPI, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create routing: %w", err)
	}

	databaseAPI, err := database.New(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create database API: %w", err)
	}

	// Create sync service
	syncService := sync.New(databaseAPI, storeAPI, options)

	// Create server
	server := &Server{
		options:       options,
		store:         storeAPI,
		routing:       routingAPI,
		database:      databaseAPI,
		syncService:   syncService,
		healthzServer: healthz.NewHealthServer(cfg.HealthCheckAddress),
		grpcServer:    grpc.NewServer(),
	}

	// Register APIs
	storetypes.RegisterStoreServiceServer(server.grpcServer, controller.NewStoreController(storeAPI, databaseAPI))
	routingtypes.RegisterRoutingServiceServer(server.grpcServer, controller.NewRoutingController(routingAPI, storeAPI))
	searchtypes.RegisterSearchServiceServer(server.grpcServer, controller.NewSearchController(databaseAPI))
	storetypes.RegisterSyncServiceServer(server.grpcServer, controller.NewSyncController(databaseAPI, options))

	// Register server
	reflection.Register(server.grpcServer)

	return server, nil
}

func (s Server) Options() types.APIOptions { return s.options }

func (s Server) Store() types.StoreAPI { return s.store }

func (s Server) Routing() types.RoutingAPI { return s.routing }

func (s Server) Database() types.DatabaseAPI { return s.database }

func (s Server) Close() {
	// Stop sync service if running
	if s.syncService != nil {
		if err := s.syncService.Stop(); err != nil {
			logger.Error("Failed to stop sync service", "error", err)
		}
	}

	s.grpcServer.GracefulStop()
}

func (s Server) start(ctx context.Context) error {
	// Bootstrap
	if err := s.bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to bootstrap server: %w", err)
	}

	// Start sync service
	if s.syncService != nil {
		if err := s.syncService.Start(ctx); err != nil {
			return fmt.Errorf("failed to start sync service: %w", err)
		}

		logger.Info("Sync service started")
	}

	// Create a listener on TCP port
	listen, err := net.Listen("tcp", s.Options().Config().ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.Options().Config().ListenAddress, err)
	}

	// Serve gRPC server in the background.
	// If the server cannot be started, exit with code 1.
	go func() {
		// Start health check server
		s.healthzServer.Start()
		s.healthzServer.SetIsReady(true)
		defer s.healthzServer.SetIsReady(false)

		logger.Info("Server starting", "address", s.Options().Config().ListenAddress)

		if err := s.grpcServer.Serve(listen); err != nil {
			logger.Error("Failed to start server", "error", err)
		}
	}()

	return nil
}

func (s Server) bootstrap(_ context.Context) error {
	// TODO: bootstrap routing and storage data by listing from storage
	// TODO: also update cache datastore
	return nil
}
