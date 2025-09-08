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
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/api/version"
	"github.com/agntcy/dir/server/authz"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/database"
	"github.com/agntcy/dir/server/publication"
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
	options            types.APIOptions
	store              types.StoreAPI
	routing            types.RoutingAPI
	database           types.DatabaseAPI
	syncService        *sync.Service
	authzService       *authz.Service
	publicationService *publication.Service
	healthzServer      *healthz.Server
	grpcServer         *grpc.Server
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

	// Load options
	options := types.NewOptions(cfg)
	serverOpts := []grpc.ServerOption{}

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

	// Create services
	syncService, err := sync.New(databaseAPI, storeAPI, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync service: %w", err)
	}

	var authzService *authz.Service
	if cfg.Authz.Enabled {
		authzService, err = authz.New(ctx, cfg.Authz)
		if err != nil {
			return nil, fmt.Errorf("failed to create authz service: %w", err)
		}

		//nolint:contextcheck
		serverOpts = append(serverOpts, authzService.GetServerOptions()...)
	}

	// Create publication service
	publicationService, err := publication.New(databaseAPI, storeAPI, routingAPI, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create publication service: %w", err)
	}

	// Create a server
	grpcServer := grpc.NewServer(serverOpts...)

	storeController, err := controller.NewStoreController(storeAPI, databaseAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to create store controller: %w", err)
	}

	// Register APIs
	storev1.RegisterStoreServiceServer(grpcServer, storeController)
	routingv1.RegisterRoutingServiceServer(grpcServer, controller.NewRoutingController(routingAPI, storeAPI, publicationService))
	routingv1.RegisterPublicationServiceServer(grpcServer, controller.NewPublicationController(databaseAPI, options))
	searchv1.RegisterSearchServiceServer(grpcServer, controller.NewSearchController(databaseAPI))
	storev1.RegisterSyncServiceServer(grpcServer, controller.NewSyncController(databaseAPI, options))
	signv1.RegisterSignServiceServer(grpcServer, controller.NewSignController(storeAPI))

	// Register server
	reflection.Register(grpcServer)

	return &Server{
		options:            options,
		store:              storeAPI,
		routing:            routingAPI,
		database:           databaseAPI,
		syncService:        syncService,
		authzService:       authzService,
		publicationService: publicationService,
		healthzServer:      healthz.NewHealthServer(cfg.HealthCheckAddress),
		grpcServer:         grpcServer,
	}, nil
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

	// Stop authz service if running
	if s.authzService != nil {
		if err := s.authzService.Stop(); err != nil {
			logger.Error("Failed to stop authz service", "error", err)
		}
	}

	// Stop publication service if running
	if s.publicationService != nil {
		if err := s.publicationService.Stop(); err != nil {
			logger.Error("Failed to stop publication service", "error", err)
		}
	}

	s.grpcServer.GracefulStop()
}

func (s Server) start(ctx context.Context) error {
	// Start sync service
	if s.syncService != nil {
		if err := s.syncService.Start(ctx); err != nil {
			return fmt.Errorf("failed to start sync service: %w", err)
		}

		logger.Info("Sync service started")
	}

	// Start publication service
	if s.publicationService != nil {
		if err := s.publicationService.Start(ctx); err != nil {
			return fmt.Errorf("failed to start publication service: %w", err)
		}

		logger.Info("Publication service started")
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
