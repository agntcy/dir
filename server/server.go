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
	"github.com/agntcy/dir/server/authn"
	authnCfg "github.com/agntcy/dir/server/authn/config"
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
	authnServices      []*authn.Service
	authzService       *authz.Service
	publicationService *publication.Service
	healthzServer      *healthz.Server
	grpcServers        map[authnCfg.AuthMode]*grpc.Server
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

	// Create publication service
	publicationService, err := publication.New(databaseAPI, storeAPI, routingAPI, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create publication service: %w", err)
	}

	authOpts := make(map[authnCfg.AuthMode][]grpc.ServerOption)

	var authnServices = []*authn.Service{}

	if cfg.Authn.Insecure.Enabled {
		authOpts[authnCfg.AuthInsecure] = []grpc.ServerOption{}
	}

	if cfg.Authn.X509.Enabled {
		authnX509Service, err := authn.New(ctx, cfg.Authn, authnCfg.AuthX509)
		if err != nil {
			return nil, fmt.Errorf("failed to create x509 authn service: %w", err)
		}

		authOpts[authnCfg.AuthX509] = append(authOpts[authnCfg.AuthX509], authnX509Service.GetServerOptions()...)
		authnServices = append(authnServices, authnX509Service)
	}

	if cfg.Authn.JWT.Enabled {
		authnJWTService, err := authn.New(ctx, cfg.Authn, authnCfg.AuthJWT)
		if err != nil {
			return nil, fmt.Errorf("failed to create jwt authn service: %w", err)
		}

		authOpts[authnCfg.AuthJWT] = append(authOpts[authnCfg.AuthJWT], authnJWTService.GetServerOptions()...)
		authnServices = append(authnServices, authnJWTService)
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

	grpcServers := make(map[authnCfg.AuthMode]*grpc.Server)

	// Create the GRPC servers
	for authMode, authOptions := range authOpts {
		serverOpts = append(serverOpts, authOptions...)
		grpcServer := grpc.NewServer(serverOpts...)

		// Register APIs
		storev1.RegisterStoreServiceServer(grpcServer, controller.NewStoreController(storeAPI, databaseAPI))
		routingv1.RegisterRoutingServiceServer(grpcServer, controller.NewRoutingController(routingAPI, storeAPI, publicationService))
		routingv1.RegisterPublicationServiceServer(grpcServer, controller.NewPublicationController(databaseAPI, options))
		searchv1.RegisterSearchServiceServer(grpcServer, controller.NewSearchController(databaseAPI))
		storev1.RegisterSyncServiceServer(grpcServer, controller.NewSyncController(databaseAPI, options))
		signv1.RegisterSignServiceServer(grpcServer, controller.NewSignController(storeAPI))

		// Register server
		reflection.Register(grpcServer)

		grpcServers[authMode] = grpcServer
	}

	return &Server{
		options:            options,
		store:              storeAPI,
		routing:            routingAPI,
		database:           databaseAPI,
		syncService:        syncService,
		authnServices:      authnServices,
		authzService:       authzService,
		publicationService: publicationService,
		healthzServer:      healthz.NewHealthServer(cfg.Authn.Insecure.HealthCheckAddress),
		grpcServers:        grpcServers,
	}, nil
}

func (s Server) Options() types.APIOptions { return s.options }

func (s Server) Store() types.StoreAPI { return s.store }

func (s Server) Routing() types.RoutingAPI { return s.routing }

func (s Server) Database() types.DatabaseAPI { return s.database }

func (s Server) Close() {
	// Stop routing service (closes GossipSub, p2p server, DHT)
	if s.routing != nil {
		if err := s.routing.Stop(); err != nil {
			logger.Error("Failed to stop routing service", "error", err)
		}
	}

	// Stop sync service if running
	if s.syncService != nil {
		if err := s.syncService.Stop(); err != nil {
			logger.Error("Failed to stop sync service", "error", err)
		}
	}

	// Stop authn services if running
	if len(s.authnServices) > 0 {
		for _, authn := range s.authnServices {
			if err := authn.Stop(); err != nil {
				logger.Error("Failed to stop authn service", "error", err)
			}
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

	for authMode, server := range s.grpcServers {
		logger.Info("Stop grpc server", "type", authMode)
		server.GracefulStop()
	}
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

	for authMode, grpcServer := range s.grpcServers {
		// Serve gRPC server in the background.
		// If the server cannot be started, exit with code 1.
		go func() {
			listenAddr := ""
			switch authMode {
			case authnCfg.AuthInsecure:
				listenAddr = s.Options().Config().Authn.Insecure.ListenAddress
			case authnCfg.AuthX509:
				listenAddr = s.Options().Config().Authn.X509.ListenAddress
			case authnCfg.AuthJWT:
				listenAddr = s.Options().Config().Authn.JWT.ListenAddress
			}

			// Create a listener on TCP port
			listen, err := net.Listen("tcp", listenAddr) //nolint:noctx
			if err != nil {
				logger.Error(fmt.Sprintf("failed to listen on %s", listenAddr), "error", err)
				return
			}

			logger.Info("Server starting", "address", listenAddr, "authMode", authMode)

			if err := grpcServer.Serve(listen); err != nil {
				logger.Error("Failed to start server", "error", err)
			}
		}()
	}

	return nil
}
