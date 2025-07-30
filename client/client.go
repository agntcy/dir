// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	storev1.StoreServiceClient
	routingv1.RoutingServiceClient
	searchv1.SearchServiceClient
	storev1.SyncServiceClient
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

	// Create client
	client, err := grpc.NewClient(
		options.config.ServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &Client{
		StoreServiceClient:   storev1.NewStoreServiceClient(client),
		RoutingServiceClient: routingv1.NewRoutingServiceClient(client),
		SearchServiceClient:  searchv1.NewSearchServiceClient(client),
		SyncServiceClient:    storev1.NewSyncServiceClient(client),
	}, nil
}
