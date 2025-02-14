// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	chunkSize = 4096 // 4KB
)

type Client struct {
	storetypes.StoreServiceClient
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
		StoreServiceClient: storetypes.NewStoreServiceClient(client),
	}, nil
}
