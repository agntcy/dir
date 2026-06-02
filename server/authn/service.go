// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/authn/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
)

var logger = logging.Logger("authn")

type Service struct {
	audiences []string
	client    *workloadapi.Client
	x509Src   x509svid.Source
	jwtSource *workloadapi.JWTSource
	bundleSrc *workloadapi.BundleSource
}

// New creates a new authentication service.
func New(ctx context.Context, cfg config.Config) (*Service, error) {
	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authn config: %w", err)
	}

	// Create a client for SPIRE Workload API
	client, err := workloadapi.New(ctx, workloadapi.WithAddr(cfg.SocketPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create Workload API client: %w", err)
	}

	// Initialize service
	service, err := newService(ctx, cfg, client)
	if err != nil {
		_ = client.Close()

		return nil, err
	}

	logger.Info("Authentication service initialized", "audiences", cfg.Audiences)

	return service, nil
}

// GetServerOptions returns gRPC server options for authentication.
func (s *Service) GetServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.Creds(
			grpccredentials.TLSServerCredentials(s.x509Src),
		),
		grpc.ChainUnaryInterceptor(JWTUnaryInterceptor(s.jwtSource, s.audiences)),
		grpc.ChainStreamInterceptor(JWTStreamInterceptor(s.jwtSource, s.audiences)),
	}
}

// Stop closes the workload API client and all sources.
func (s *Service) Stop() error {
	if s.jwtSource != nil {
		if err := s.jwtSource.Close(); err != nil {
			logger.Error("Failed to close JWT source", "error", err)
		}
	}

	if s.bundleSrc != nil {
		if err := s.bundleSrc.Close(); err != nil {
			logger.Error("Failed to close Bundle source", "error", err)
		}
	}

	if s.client != nil {
		if err := s.client.Close(); err != nil {
			logger.Error("Failed to close Workload API client", "error", err)
		}
	}

	return nil
}

// newService sets up the authentication service by creating necessary sources.
func newService(ctx context.Context, cfg config.Config, client *workloadapi.Client) (*Service, error) {
	// Create X.509 source for server's TLS certificate
	x509Src, err := workloadapi.NewX509Source(ctx, workloadapi.WithClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create X509 source: %w", err)
	}

	// Create bundle source for trust verification
	bundleSrc, err := workloadapi.NewBundleSource(ctx, workloadapi.WithClient(client))
	if err != nil {
		_ = x509Src.Close()

		return nil, fmt.Errorf("failed to create Bundle source: %w", err)
	}

	// Create JWT source for validating client JWT-SVIDs
	jwtSource, err := workloadapi.NewJWTSource(ctx, workloadapi.WithClient(client))
	if err != nil {
		_ = x509Src.Close()
		_ = bundleSrc.Close()

		return nil, fmt.Errorf("failed to create JWT source: %w", err)
	}

	return &Service{
		audiences: cfg.Audiences,
		client:    client,
		x509Src:   x509Src,
		bundleSrc: bundleSrc,
		jwtSource: jwtSource,
	}, nil
}
