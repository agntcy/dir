// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"errors"
	"fmt"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/server/config"
	"github.com/agntcy/dir/runtime/server/database"
	grpcserver "github.com/agntcy/dir/runtime/server/grpc"
	"github.com/agntcy/dir/runtime/store"
	storetypes "github.com/agntcy/dir/runtime/store/types"
	"github.com/agntcy/dir/runtime/utils"
	"google.golang.org/grpc"
)

// Option configures server component behavior.
type Option func(*options) error

type options struct {
	cfg    *config.Config
	store  storetypes.StoreReader
	logger *utils.Logger
}

// WithConfig sets store configuration.
func WithConfig(cfg *config.Config) Option {
	return func(o *options) error {
		if cfg == nil {
			return fmt.Errorf("config cannot be nil")
		}

		o.cfg = cfg

		return nil
	}
}

// WithStore uses an existing store instance.
func WithStore(store storetypes.StoreReader) Option {
	return func(o *options) error {
		if store == nil {
			return fmt.Errorf("store cannot be nil")
		}

		o.store = store

		return nil
	}
}

// WithLogger sets the logger used by server.
func WithLogger(logger *utils.Logger) Option {
	return func(o *options) error {
		if logger == nil {
			return fmt.Errorf("logger cannot be nil")
		}

		o.logger = logger

		return nil
	}
}

// Server encapsulates runtime server internals that can be registered on a caller-managed gRPC server.
type Server struct {
	db         *database.Database
	store      storetypes.StoreReader
	closeStore bool
	logger     *utils.Logger
}

// New creates a server component with dependencies configured through options.
func New(opts ...Option) (*Server, error) {
	o := &options{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if err := opt(o); err != nil {
			return nil, fmt.Errorf("invalid option: %w", err)
		}
	}

	// Create logger if not provided
	if o.logger == nil {
		o.logger = utils.NewLogger("runtime", "server")
	}

	// Create config if not provided
	if o.cfg == nil {
		cfg, err := config.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}

		o.cfg = cfg
	}

	// Create store if not provided
	closeStore := false

	if o.store == nil {
		storeReader, err := store.New(o.cfg.Store)
		if err != nil {
			return nil, fmt.Errorf("failed to create store: %w", err)
		}

		o.store = storeReader
		closeStore = true
	}

	// Create database
	db, err := database.NewDatabase(o.store)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return &Server{
		db:         db,
		store:      o.store,
		closeStore: closeStore,
		logger:     o.logger,
	}, nil
}

func (s *Server) Register(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc registrar cannot be nil")
	}

	discoveryServer := grpcserver.NewDiscovery(s.db)
	runtimev1.RegisterDiscoveryServiceServer(registrar, discoveryServer)

	s.logger.Info("registered runtime discovery gRPC service")

	return nil
}

// Close releases internally created dependencies.
func (s *Server) Close() error {
	// Aggregate errors from closing dependencies, if any.
	var aggErr error

	// Close database
	if err := s.db.Close(); err != nil {
		aggErr = errors.Join(aggErr, fmt.Errorf("failed to close database: %w", err))
	}

	// Close store if owned
	if s.closeStore {
		if err := s.store.Close(); err != nil {
			aggErr = errors.Join(aggErr, fmt.Errorf("failed to close store: %w", err))
		}
	}

	return aggErr
}
