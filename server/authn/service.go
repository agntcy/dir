// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/authn/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
)

var logger = logging.Logger("authn")

// Service manages authentication using SPIFFE (X.509 or JWT).
type Service struct {
	mode      config.AuthMode
	audiences []string
	client    *workloadapi.Client
	jwtSource *workloadapi.JWTSource
	x509Src   *workloadapi.X509Source
	bundleSrc *workloadapi.BundleSource
}

// New creates a new authentication service (JWT or X.509 based on config).
func New(ctx context.Context, cfg config.Config) (*Service, error) {
	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authn config: %w", err)
	}

	// Create a client for SPIRE Workload API
	client, err := workloadapi.New(ctx, workloadapi.WithAddr(cfg.SocketPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create workload API client: %w", err)
	}

	service := &Service{
		mode:   cfg.Mode,
		client: client,
	}

	// Initialize based on authentication mode
	switch cfg.Mode {
	case config.AuthModeJWT:
		if err := service.initJWT(ctx, cfg); err != nil {
			_ = client.Close()

			return nil, err
		}

		logger.Info("JWT authentication service initialized", "audiences", cfg.Audiences)

	case config.AuthModeX509:
		if err := service.initX509(ctx); err != nil {
			_ = client.Close()

			return nil, err
		}

		logger.Info("X.509 authentication service initialized")

	default:
		_ = client.Close()

		return nil, fmt.Errorf("unsupported auth mode: %s", cfg.Mode)
	}

	return service, nil
}

// initJWT initializes JWT authentication components.
// For JWT mode, the server presents its X.509-SVID via TLS (for server authentication and encryption),
// while clients authenticate using JWT-SVIDs. This follows the official SPIFFE JWT pattern.
func (s *Service) initJWT(ctx context.Context, cfg config.Config) error {
	// Create X.509 source for server's TLS certificate
	x509Src, err := workloadapi.NewX509Source(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		return fmt.Errorf("failed to create X509 source: %w", err)
	}

	// Create bundle source for trust verification
	bundleSrc, err := workloadapi.NewBundleSource(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		_ = x509Src.Close()

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	// Create JWT source for validating client JWT-SVIDs
	jwtSource, err := workloadapi.NewJWTSource(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		_ = x509Src.Close()
		_ = bundleSrc.Close()

		return fmt.Errorf("failed to create JWT source: %w", err)
	}

	s.x509Src = x509Src
	s.bundleSrc = bundleSrc
	s.jwtSource = jwtSource
	s.audiences = cfg.Audiences

	return nil
}

// initX509 initializes X.509 authentication components.
func (s *Service) initX509(ctx context.Context) error {
	// Create a new X509 source which periodically refetches X509-SVIDs and X.509 bundles
	x509Src, err := workloadapi.NewX509Source(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		return fmt.Errorf("failed to create X509 source: %w", err)
	}

	// Create a new Bundle source which periodically refetches SPIFFE bundles.
	// Required when running Federation.
	bundleSrc, err := workloadapi.NewBundleSource(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		_ = x509Src.Close()

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	s.x509Src = x509Src
	s.bundleSrc = bundleSrc

	return nil
}

// GetServerOptions returns gRPC server options for authentication.
func (s *Service) GetServerOptions() []grpc.ServerOption {
	switch s.mode {
	case config.AuthModeJWT:
		// JWT mode: Server presents X.509-SVID via TLS, clients authenticate with JWT-SVID
		return []grpc.ServerOption{
			grpc.Creds(
				grpccredentials.TLSServerCredentials(s.x509Src),
			),
			grpc.ChainUnaryInterceptor(JWTUnaryInterceptor(s.jwtSource, s.audiences)),
			grpc.ChainStreamInterceptor(JWTStreamInterceptor(s.jwtSource, s.audiences)),
		}

	case config.AuthModeX509:
		return []grpc.ServerOption{
			grpc.Creds(
				grpccredentials.MTLSServerCredentials(s.x509Src, s.bundleSrc, tlsconfig.AuthorizeAny()),
			),
			grpc.ChainUnaryInterceptor(X509UnaryInterceptor()),
			grpc.ChainStreamInterceptor(X509StreamInterceptor()),
		}

	default:
		logger.Error("Unsupported auth mode", "mode", s.mode)

		return []grpc.ServerOption{}
	}
}

// Stop closes the workload API client and all sources.
//
//nolint:wrapcheck
func (s *Service) Stop() error {
	if s.jwtSource != nil {
		if err := s.jwtSource.Close(); err != nil {
			logger.Error("Failed to close JWT source", "error", err)
		}
	}

	if s.x509Src != nil {
		if err := s.x509Src.Close(); err != nil {
			logger.Error("Failed to close X509 source", "error", err)
		}
	}

	if s.bundleSrc != nil {
		if err := s.bundleSrc.Close(); err != nil {
			logger.Error("Failed to close bundle source", "error", err)
		}
	}

	return s.client.Close()
}
