// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
	searchtypesv1alpha2 "github.com/agntcy/dir/api/search/v1alpha2"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	chunkSize = 4096 // 4KB
)

type Client struct {
	storetypes.StoreServiceClient
	routingtypes.RoutingServiceClient
	searchtypesv1alpha2.SearchServiceClient
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
		StoreServiceClient:   storetypes.NewStoreServiceClient(client),
		RoutingServiceClient: routingtypes.NewRoutingServiceClient(client),
		SearchServiceClient:  searchtypesv1alpha2.NewSearchServiceClient(client),
	}, nil
}
