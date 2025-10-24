// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"io"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
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
	eventsv1.EventServiceClient

	config     *Config
	authClient *workloadapi.Client
	conn       *grpc.ClientConn

	// SPIFFE sources for cleanup
	bundleSrc io.Closer
	x509Src   io.Closer
	jwtSource io.Closer
}

func New(ctx context.Context, opts ...Option) (*Client, error) {
	// Add auth options with provided context
	opts = append(opts, withAuth(ctx))

	// Load options
	options := &options{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, fmt.Errorf("failed to load options: %w", err)
		}
	}

	// Create gRPC client connection
	conn, err := grpc.NewClient(options.config.ServerAddress, options.authOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &Client{
		StoreServiceClient:   storev1.NewStoreServiceClient(conn),
		RoutingServiceClient: routingv1.NewRoutingServiceClient(conn),
		SearchServiceClient:  searchv1.NewSearchServiceClient(conn),
		SyncServiceClient:    storev1.NewSyncServiceClient(conn),
		SignServiceClient:    signv1.NewSignServiceClient(conn),
		EventServiceClient:   eventsv1.NewEventServiceClient(conn),
		config:               options.config,
		authClient:           options.authClient,
		conn:                 conn,
		bundleSrc:            options.bundleSrc,
		x509Src:              options.x509Src,
		jwtSource:            options.jwtSource,
	}, nil
}

func (c *Client) Close() error {
	var errs []error

	// Close SPIFFE sources first (they may be using authClient)
	if c.jwtSource != nil {
		if err := c.jwtSource.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close JWT source: %w", err))
		}
	}

	if c.x509Src != nil {
		if err := c.x509Src.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close X.509 source: %w", err))
		}
	}

	if c.bundleSrc != nil {
		if err := c.bundleSrc.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close bundle source: %w", err))
		}
	}

	// Close auth client
	if c.authClient != nil {
		if err := c.authClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close auth client: %w", err))
		}
	}

	// Close gRPC connection last
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close gRPC connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("client close errors: %v", errs)
	}

	return nil
}
