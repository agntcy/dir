// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Option func(*options) error

// TODO: options need to be granular per key rather than for full config.
type options struct {
	config     *Config
	authOpts   []grpc.DialOption
	authClient *workloadapi.Client
}

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

func withAuth(ctx context.Context) Option {
	return func(o *options) error {
		// Use insecure access in case SpiffeSocketPath is not set
		if o.config.SpiffeSocketPath == "" {
			o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

			return nil
		}

		// Create SPIFFE client
		client, err := workloadapi.New(ctx, workloadapi.WithAddr(o.config.SpiffeSocketPath))
		if err != nil {
			return fmt.Errorf("failed to create SPIFFE client: %w", err)
		}

		// Create SPIFFE x509 services
		x509Src, err := workloadapi.NewX509Source(ctx, workloadapi.WithClient(client))
		if err != nil {
			_ = client.Close()

			return fmt.Errorf("failed to create x509 source: %w", err)
		}

		// Create SPIFFE bundle services
		bundleSrc, err := workloadapi.NewBundleSource(ctx, workloadapi.WithClient(client))
		if err != nil {
			_ = client.Close()

			return fmt.Errorf("failed to create bundle source: %w", err)
		}

		// Add auth options to the client
		o.authClient = client
		o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(
			grpccredentials.MTLSClientCredentials(x509Src, bundleSrc, tlsconfig.AuthorizeAny()),
		))

		return nil
	}
}
