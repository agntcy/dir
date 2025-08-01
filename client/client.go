// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
	searchtypesv1alpha2 "github.com/agntcy/dir/api/search/v1alpha2"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
)

type Client struct {
	storetypes.StoreServiceClient
	routingtypes.RoutingServiceClient
	searchtypesv1alpha2.SearchServiceClient
	storetypes.SyncServiceClient
}

type options struct {
	config *Config
}

type Option func(*options) error

func WithEnvConfig() Option {
	return func(opts *options) error {
		var err error
		opts.config, err = LoadConfig()

		return err
	}
}

func WithConfig(config *Config) Option {
	return func(opts *options) error {
		opts.config = config

		return nil
	}
}

func New(opts ...Option) (*Client, error) {
	// Load options
	options := &options{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, fmt.Errorf("failed to load options: %w", err)
		}
	}

	// Create context for SPIFFE
	ctx := context.Background()

	// Create client transport options
	var clientOpts []grpc.DialOption

	// Create SPIFFE mTLS services if configured
	if options.config.SpiffeSocketPath != "" {
		x509Src, err := workloadapi.NewX509Source(ctx,
			workloadapi.WithClientOptions(
				workloadapi.WithAddr(options.config.SpiffeSocketPath),
				//workloadapi.WithLogger(logger.Std),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch svid: %w", err)
		}

		bundleSrc, err := workloadapi.NewBundleSource(ctx,
			workloadapi.WithClientOptions(
				workloadapi.WithAddr(options.config.SpiffeSocketPath),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch trust bundle: %w", err)
		}

		trustDomain, err := spiffeid.TrustDomainFromString(options.config.SpiffeTrustDomain)
		if err != nil {
			return nil, fmt.Errorf("failed to parse trust domain: %w", err)
		}

		// Add client options for SPIFFE mTLS
		clientOpts = append(clientOpts, grpc.WithTransportCredentials(
			grpccredentials.MTLSClientCredentials(x509Src, bundleSrc, tlsconfig.AuthorizeMemberOf(trustDomain)),
		))
	} else {
		clientOpts = append(clientOpts, grpc.WithInsecure())
	}

	// Create client
	client, err := grpc.NewClient(options.config.ServerAddress, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &Client{
		StoreServiceClient:   storetypes.NewStoreServiceClient(client),
		RoutingServiceClient: routingtypes.NewRoutingServiceClient(client),
		SearchServiceClient:  searchtypesv1alpha2.NewSearchServiceClient(client),
		SyncServiceClient:    storetypes.NewSyncServiceClient(client),
	}, nil
}
