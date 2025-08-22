// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	storev1.StoreServiceClient
	routingv1.RoutingServiceClient
	searchv1.SearchServiceClient
	storev1.SyncServiceClient
	signv1.SignServiceClient
	config *Config
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
				// workloadapi.WithLogger(logger.Std),
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

		// Add client options for SPIFFE mTLS
		clientOpts = append(clientOpts, grpc.WithTransportCredentials(
			grpccredentials.MTLSClientCredentials(x509Src, bundleSrc, tlsconfig.AuthorizeAny()),
		))
	} else {
		clientOpts = append(clientOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Create client
	client, err := grpc.NewClient(options.config.ServerAddress, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &Client{
		StoreServiceClient:   storev1.NewStoreServiceClient(client),
		RoutingServiceClient: routingv1.NewRoutingServiceClient(client),
		SearchServiceClient:  searchv1.NewSearchServiceClient(client),
		SyncServiceClient:    storev1.NewSyncServiceClient(client),
		SignServiceClient:    signv1.NewSignServiceClient(client),
		config:               options.config,
	}, nil
}
