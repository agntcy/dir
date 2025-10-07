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
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
)

type Client struct {
	storev1.StoreServiceClient
	routingv1.RoutingServiceClient
	searchv1.SearchServiceClient
	storev1.SyncServiceClient
	signv1.SignServiceClient

	config     *Config
	authClient *workloadapi.Client
}

func New(opts ...Option) (*Client, error) {
	// Add auth options
	opts = append(opts, withAuth(context.Background()))

	// Load options
	options := &options{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, fmt.Errorf("failed to load options: %w", err)
		}
	}

	// Create client
	client, err := grpc.NewClient(options.config.ServerAddress, options.authOpts...)
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
		authClient:           options.authClient,
	}, nil
}

func (c *Client) Close() error {
	// Close auth client if it exists
	if c.authClient != nil {
		//nolint:wrapcheck
		return c.authClient.Close()
	}

	return nil
}
