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
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/api/version"
	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/server/authz"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/database"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/healthcheck"
	grpclogging "github.com/agntcy/dir/server/middleware/logging"
	grpcratelimit "github.com/agntcy/dir/server/middleware/ratelimit"
	grpcrecovery "github.com/agntcy/dir/server/middleware/recovery"
	"github.com/agntcy/dir/server/publication"
	"github.com/agntcy/dir/server/routing"
	"github.com/agntcy/dir/server/store"
	"github.com/agntcy/dir/server/sync"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

const (
	// bytesToMB is the conversion factor from bytes to megabytes.
	bytesToMB = 1024 * 1024
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
	eventService       *events.Service
	syncService        *sync.Service
	authnService       *authn.Service
	authzService       *authz.Service
	publicationService *publication.Service
	health             *healthcheck.Checker
	grpcServer         *grpc.Server
}

// buildConnectionOptions creates gRPC server options for connection management.
// These options configure connection limits, keepalive parameters, and message size limits
// to prevent resource exhaustion and detect dead connections.
//
// Connection management is applied BEFORE all interceptors to ensure limits are enforced
// at the lowest level, protecting all other server components.
func buildConnectionOptions(cfg config.ConnectionConfig) []grpc.ServerOption {
	opts := []grpc.ServerOption{
		// Connection limits - prevent resource monopolization
		grpc.MaxConcurrentStreams(cfg.MaxConcurrentStreams),
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
		grpc.ConnectionTimeout(cfg.ConnectionTimeout),

		// Keepalive parameters - detect dead connections and rotate aged connections
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     cfg.Keepalive.MaxConnectionIdle,
			MaxConnectionAge:      cfg.Keepalive.MaxConnectionAge,
			MaxConnectionAgeGrace: cfg.Keepalive.MaxConnectionAgeGrace,
			Time:                  cfg.Keepalive.Time,
			Timeout:               cfg.Keepalive.Timeout,
		}),

		// Keepalive enforcement policy - prevent client abuse
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             cfg.Keepalive.MinTime,
			PermitWithoutStream: cfg.Keepalive.PermitWithoutStream,
		}),
	}

	logger.Info("Connection management configured",
		"max_concurrent_streams", cfg.MaxConcurrentStreams,
		"max_recv_msg_size_mb", cfg.MaxRecvMsgSize/bytesToMB,
		"max_send_msg_size_mb", cfg.MaxSendMsgSize/bytesToMB,
		"connection_timeout", cfg.ConnectionTimeout,
		"max_connection_idle", cfg.Keepalive.MaxConnectionIdle,
		"max_connection_age", cfg.Keepalive.MaxConnectionAge,
		"keepalive_time", cfg.Keepalive.Time,
		"keepalive_timeout", cfg.Keepalive.Timeout,
	)

	return opts
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
	defer server.Close(ctx)

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

	// Add connection management options FIRST (lowest level - applies to all connections)
	// This must be before interceptors to ensure connection limits protect all server components
	connConfig := cfg.Connection.WithDefaults()
	connectionOpts := buildConnectionOptions(connConfig)
	serverOpts = append(serverOpts, connectionOpts...)

	// Add panic recovery interceptors (after connection management, before other interceptors)
	// This prevents server crashes from panics in handlers or other interceptors
	serverOpts = append(serverOpts, grpcrecovery.ServerOptions()...)

	// Add rate limiting interceptors (after recovery, before logging and auth)
	// This protects authentication and other downstream processes from DDoS attacks
	if cfg.RateLimit.Enabled {
		rateLimitOpts, err := grpcratelimit.ServerOptions(&cfg.RateLimit)
		if err != nil {
			return nil, fmt.Errorf("failed to create rate limit interceptors: %w", err)
		}

		serverOpts = append(serverOpts, rateLimitOpts...)

		logger.Info("Rate limiting enabled",
			"global_rps", cfg.RateLimit.GlobalRPS,
			"per_client_rps", cfg.RateLimit.PerClientRPS,
		)
	}

	// Add gRPC logging interceptors (after recovery and rate limiting, before auth/authz)
	grpcLogger := logging.Logger("grpc")
	loggingOpts := grpclogging.ServerOptions(grpcLogger, cfg.Logging.Verbose)
	serverOpts = append(serverOpts, loggingOpts...)

	// Create event service first (so other services can emit events)
	eventService := events.New()
	safeEventBus := events.NewSafeEventBus(eventService.Bus())

	// Add event bus to options for other services
	options = options.WithEventBus(safeEventBus)

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

	// Create JWT authentication service if enabled
	var authnService *authn.Service
	if cfg.Authn.Enabled {
		authnService, err = authn.New(ctx, cfg.Authn)
		if err != nil {
			return nil, fmt.Errorf("failed to create authn service: %w", err)
		}

		//nolint:contextcheck
		serverOpts = append(serverOpts, authnService.GetServerOptions()...)
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

	// Create health checker
	healthChecker := healthcheck.New()

	// Register APIs
	eventsv1.RegisterEventServiceServer(grpcServer, controller.NewEventsController(eventService))
	storev1.RegisterStoreServiceServer(grpcServer, controller.NewStoreController(storeAPI, databaseAPI, options.EventBus()))
	routingv1.RegisterRoutingServiceServer(grpcServer, controller.NewRoutingController(routingAPI, storeAPI, publicationService))
	routingv1.RegisterPublicationServiceServer(grpcServer, controller.NewPublicationController(databaseAPI, options))
	searchv1.RegisterSearchServiceServer(grpcServer, controller.NewSearchController(databaseAPI))
	storev1.RegisterSyncServiceServer(grpcServer, controller.NewSyncController(databaseAPI, options))
	signv1.RegisterSignServiceServer(grpcServer, controller.NewSignController(storeAPI))

	// Register health service
	healthChecker.Register(grpcServer)

	// Register reflection service
	reflection.Register(grpcServer)

	return &Server{
		options:            options,
		store:              storeAPI,
		routing:            routingAPI,
		database:           databaseAPI,
		eventService:       eventService,
		syncService:        syncService,
		authnService:       authnService,
		authzService:       authzService,
		publicationService: publicationService,
		health:             healthChecker,
		grpcServer:         grpcServer,
	}, nil
}

func (s Server) Options() types.APIOptions { return s.options }

func (s Server) Store() types.StoreAPI { return s.store }

func (s Server) Routing() types.RoutingAPI { return s.routing }

func (s Server) Database() types.DatabaseAPI { return s.database }

func (s Server) Close(ctx context.Context) {
	// Stop health check monitoring
	if s.health != nil {
		stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second) //nolint:mnd
		defer cancel()

		if err := s.health.Stop(stopCtx); err != nil {
			logger.Error("Failed to stop health check service", "error", err)
		}
	}

	// Stop event service
	if s.eventService != nil {
		if err := s.eventService.Stop(); err != nil {
			logger.Error("Failed to stop event service", "error", err)
		}
	}

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

	// Stop authn service if running
	if s.authnService != nil {
		if err := s.authnService.Stop(); err != nil {
			logger.Error("Failed to stop authn service", "error", err)
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
	listen, err := net.Listen("tcp", s.Options().Config().ListenAddress) //nolint:noctx
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.Options().Config().ListenAddress, err)
	}

	// Add readiness checks
	s.health.AddReadinessCheck("database", s.database.IsReady)
	s.health.AddReadinessCheck("sync", s.syncService.IsReady)
	s.health.AddReadinessCheck("publication", s.publicationService.IsReady)
	s.health.AddReadinessCheck("store", s.store.IsReady)
	s.health.AddReadinessCheck("routing", s.routing.IsReady)

	// Start health check monitoring
	if err := s.health.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health check monitoring: %w", err)
	}

	// Serve gRPC server in the background
	go func() {
		logger.Info("Server starting", "address", s.Options().Config().ListenAddress)

		if err := s.grpcServer.Serve(listen); err != nil {
			logger.Error("Failed to start server", "error", err)
		}
	}()

	return nil
}
