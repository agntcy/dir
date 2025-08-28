// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/authz/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
)

var logger = logging.Logger("authz")

type Service struct {
	authorizer *Authorizer
	client     *workloadapi.Client
	x509Src    *workloadapi.X509Source
	bundleSrc  *workloadapi.BundleSource
}

func New(ctx context.Context, cfg config.Config) (*Service, error) {
	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authz config: %w", err)
	}

	// Create authorizer
	authorizer, err := NewAuthorizer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %w", err)
	}

	// Create a client for SPIRE Workload API
	client, err := workloadapi.New(ctx, workloadapi.WithAddr(cfg.SocketPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create workload API client: %w", err)
	}

	// Create a new X509 source which periodically refetches X509-SVIDs and X.509 bundles
	x509Src, err := workloadapi.NewX509Source(ctx, workloadapi.WithClient(client))
	if err != nil {
		_ = client.Close()

		return nil, fmt.Errorf("failed to fetch svid: %w", err)
	}

	// Create a new Bundle source which periodically refetches SPIFFE bundles.
	// Required when running Federation.
	bundleSrc, err := workloadapi.NewBundleSource(ctx, workloadapi.WithClient(client))
	if err != nil {
		_ = client.Close()

		return nil, fmt.Errorf("failed to fetch trust bundle: %w", err)
	}

	return &Service{
		authorizer: authorizer,
		client:     client,
		x509Src:    x509Src,
		bundleSrc:  bundleSrc,
	}, nil
}

func (s *Service) GetServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.Creds(
			grpccredentials.MTLSServerCredentials(s.x509Src, s.bundleSrc, tlsconfig.AuthorizeAny()),
		),
		grpc.ChainUnaryInterceptor(UnaryInterceptorFor(NewInterceptor(s.authorizer))),
		grpc.ChainStreamInterceptor(StreamInterceptorFor(NewInterceptor(s.authorizer))),
	}
}

//nolint:wrapcheck
func (s *Service) Stop() error {
	return s.client.Close()
}
