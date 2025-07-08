// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
)

type Client struct {
	storev1.StoreServiceClient
	routingv1.RoutingServiceClient
	searchv1.SearchServiceClient
	storev1.SyncServiceClient

	closeFn func() error
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

func (c *Client) Close() error {
	if c.closeFn == nil {
		return nil
	}

	return c.closeFn()
}

func New(opts ...Option) (*Client, error) {
	// Load options
	options := &options{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, fmt.Errorf("failed to load options: %w", err)
		}
	}

	// Create a `workloadapi.X509Source`, it will connect to Workload API using provided socket path
	// If socket path is not defined using `workloadapi.SourceOption`, value from environment variable `SPIFFE_ENDPOINT_SOCKET` is used.
	source, err := workloadapi.NewX509Source(context.Background(), workloadapi.WithClientOptions(workloadapi.WithAddr(options.config.SpiffeWorkloadAddress)))
	if err != nil {
		return nil, fmt.Errorf("unable to create X509Source: %w", err)
	}

	// Allowed SPIFFE ID
	clientDomain := spiffeid.RequireTrustDomainFromString("spiffe://example.org")

	// Create client
	client, err := grpc.NewClient(
		options.config.ServerAddress,
		grpc.WithTransportCredentials(
			grpccredentials.MTLSClientCredentials(source, source, tlsconfig.AuthorizeMemberOf(clientDomain)),
		),
	)
	if err != nil {
		defer source.Close()

		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &Client{
		StoreServiceClient:   storev1.NewStoreServiceClient(client),
		RoutingServiceClient: routingv1.NewRoutingServiceClient(client),
		SearchServiceClient:  searchv1.NewSearchServiceClient(client),
		SyncServiceClient:    storev1.NewSyncServiceClient(client),
		closeFn:              source.Close,
	}, nil
}
