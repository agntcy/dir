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
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/database"
	"github.com/agntcy/dir/server/routing"
	"github.com/agntcy/dir/server/store"
	"github.com/agntcy/dir/server/sync"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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
	syncService, err := sync.New(databaseAPI, storeAPI, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync service: %w", err)
	}

	// Create server transport options
	var serverOpts []grpc.ServerOption

	// Create SPIFFE mTLS services if configured
	if cfg.Authz.SocketPath != "" {
		x509Src, err := workloadapi.NewX509Source(ctx,
			workloadapi.WithClientOptions(
				workloadapi.WithAddr(cfg.Authz.SocketPath),
				// workloadapi.WithLogger(logger.Std),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch svid: %w", err)
		}

		bundleSrc, err := workloadapi.NewBundleSource(ctx,
			workloadapi.WithClientOptions(
				workloadapi.WithAddr(cfg.Authz.SocketPath),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch trust bundle: %w", err)
		}

		// Add server options for SPIFFE mTLS
		//nolint:contextcheck
		serverOpts = append(serverOpts,
			grpc.Creds(
				grpccredentials.MTLSServerCredentials(x509Src, bundleSrc, tlsconfig.AuthorizeAny()),
			),
			grpc.ChainUnaryInterceptor(unaryInterceptorFor(authInterceptor)),
			grpc.ChainStreamInterceptor(streamInterceptorFor(authInterceptor)),
		)
	}

	// Create a server
	grpcServer := grpc.NewServer(serverOpts...)

	// Register APIs
	storev1.RegisterStoreServiceServer(grpcServer, controller.NewStoreController(storeAPI, databaseAPI))
	routingv1.RegisterRoutingServiceServer(grpcServer, controller.NewRoutingController(routingAPI, storeAPI))
	searchv1.RegisterSearchServiceServer(grpcServer, controller.NewSearchController(databaseAPI))
	storev1.RegisterSyncServiceServer(grpcServer, controller.NewSyncController(databaseAPI, options))
	signv1.RegisterSignServiceServer(grpcServer, controller.NewSignController(storeAPI))

	// Register server
	reflection.Register(grpcServer)

	return &Server{
		options:       options,
		store:         storeAPI,
		routing:       routingAPI,
		database:      databaseAPI,
		syncService:   syncService,
		healthzServer: healthz.NewHealthServer(cfg.HealthCheckAddress),
		grpcServer:    grpcServer,
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

func authInterceptor(ctx context.Context) error {
	sid, ok := grpccredentials.PeerIDFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing peer ID") //nolint:wrapcheck
	}

	logger.Debug("Authenticated user", "id", sid)

	return nil
}

// TODO: this can be moved to utils and expanded.
func unaryInterceptorFor(fn func(context.Context) error) func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := fn(ctx); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// TODO: this can be moved to utils and expanded.
func streamInterceptorFor(fn func(context.Context) error) func(any, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
	return func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := fn(ss.Context()); err != nil {
			return err
		}

		return handler(srv, ss)
	}
}
