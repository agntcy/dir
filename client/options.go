// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
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
		// Use insecure access in case SpiffeSocketPath is not set or no auth mode specified
		if o.config.SpiffeSocketPath == "" || o.config.AuthMode == "" {
			o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

			return nil
		}

		// Create SPIFFE client
		client, err := workloadapi.New(ctx, workloadapi.WithAddr(o.config.SpiffeSocketPath))
		if err != nil {
			return fmt.Errorf("failed to create SPIFFE client: %w", err)
		}

		o.authClient = client

		switch o.config.AuthMode {
		case "jwt":
			return o.setupJWTAuth(ctx, client)
		case "x509":
			return o.setupX509Auth(ctx, client)
		default:
			_ = client.Close()

			return fmt.Errorf("unsupported auth mode: %s (supported: 'jwt', 'x509')", o.config.AuthMode)
		}
	}
}

func (o *options) setupJWTAuth(ctx context.Context, client *workloadapi.Client) error {
	// Validate JWT audience is set
	if o.config.JWTAudience == "" {
		_ = client.Close()

		return errors.New("JWT audience is required for JWT authentication")
	}

	// Create bundle source for verifying server's TLS certificate (X.509-SVID)
	bundleSrc, err := workloadapi.NewBundleSource(ctx, workloadapi.WithClient(client))
	if err != nil {
		_ = client.Close()

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	// Create JWT source for fetching JWT-SVIDs
	jwtSource, err := workloadapi.NewJWTSource(ctx, workloadapi.WithClient(client))
	if err != nil {
		_ = client.Close()
		_ = bundleSrc.Close()

		return fmt.Errorf("failed to create JWT source: %w", err)
	}

	// Use TLS for transport security (server presents X.509-SVID)
	// Client authenticates with JWT-SVID via PerRPCCredentials
	o.authOpts = append(o.authOpts,
		grpc.WithTransportCredentials(
			grpccredentials.TLSClientCredentials(bundleSrc, tlsconfig.AuthorizeAny()),
		),
		grpc.WithPerRPCCredentials(newJWTCredentials(jwtSource, o.config.JWTAudience)),
	)

	return nil
}

func (o *options) setupX509Auth(ctx context.Context, client *workloadapi.Client) error {
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
	o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(
		grpccredentials.MTLSClientCredentials(x509Src, bundleSrc, tlsconfig.AuthorizeAny()),
	))

	return nil
}
