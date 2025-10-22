// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"

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

	// SPIFFE sources for cleanup
	bundleSrc io.Closer
	x509Src   io.Closer
	jwtSource io.Closer
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
		// Validate config exists before dereferencing
		if o.config == nil {
			return errors.New("config is required: use WithConfig() or WithEnvConfig()")
		}

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
			//nolint:contextcheck // SPIFFE sources need context.Background() for long lifetime, not init ctx
			return o.setupJWTAuth(client)
		case "x509":
			//nolint:contextcheck // SPIFFE sources need context.Background() for long lifetime, not init ctx
			return o.setupX509Auth(client)
		default:
			_ = client.Close()

			return fmt.Errorf("unsupported auth mode: %s (supported: 'jwt', 'x509')", o.config.AuthMode)
		}
	}
}

func (o *options) setupJWTAuth(client *workloadapi.Client) error {
	// Validate JWT audience is set
	if o.config.JWTAudience == "" {
		_ = client.Close()

		return errors.New("JWT audience is required for JWT authentication")
	}

	// Create bundle source for verifying server's TLS certificate (X.509-SVID)
	// Note: Use context.Background() for long-running sources that must live as long as the client
	bundleSrc, err := workloadapi.NewBundleSource(context.Background(), workloadapi.WithClient(client))
	if err != nil {
		_ = client.Close()

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	o.bundleSrc = bundleSrc // Store for cleanup

	// Create JWT source for fetching JWT-SVIDs
	// Note: Use context.Background() for long-running sources that must live as long as the client
	jwtSource, err := workloadapi.NewJWTSource(context.Background(), workloadapi.WithClient(client))
	if err != nil {
		_ = client.Close()
		_ = bundleSrc.Close()

		return fmt.Errorf("failed to create JWT source: %w", err)
	}

	o.jwtSource = jwtSource // Store for cleanup

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

func (o *options) setupX509Auth(client *workloadapi.Client) error {
	// Create SPIFFE x509 services
	// Note: Use context.Background() for long-running sources that must live as long as the client
	x509Src, err := workloadapi.NewX509Source(context.Background(), workloadapi.WithClient(client))
	if err != nil {
		_ = client.Close()

		return fmt.Errorf("failed to create x509 source: %w", err)
	}

	o.x509Src = x509Src // Store for cleanup

	// Create SPIFFE bundle services
	// Note: Use context.Background() for long-running sources that must live as long as the client
	bundleSrc, err := workloadapi.NewBundleSource(context.Background(), workloadapi.WithClient(client))
	if err != nil {
		_ = client.Close()
		_ = x509Src.Close() // Fix Issue #4: Close x509Src on error

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	o.bundleSrc = bundleSrc // Store for cleanup

	// Add auth options to the client
	o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(
		grpccredentials.MTLSClientCredentials(x509Src, bundleSrc, tlsconfig.AuthorizeAny()),
	))

	return nil
}
