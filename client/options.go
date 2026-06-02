// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/agntcy/dir/utils/logging"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var authLogger = logging.Logger("client.auth")

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

		// Setup authentication based on AuthMode
		switch o.config.AuthMode {
		case "jwt":
			// NOTE: jwt source must live for the entire client lifetime, not just the initialization phase
			return o.setupJWTAuth(ctx) //nolint:contextcheck
		case "tls":
			return o.setupTlsAuth(ctx)
		case "oidc":
			return o.setupOIDCAuth(ctx)
		case "insecure", "none", "":
			// Insecure/none/empty auth mode - try auto-detection first, fallback to insecure
			return o.setupAutoDetectAuth(ctx)
		default:
			// Invalid auth mode specified - return error to prevent silent security issues
			return fmt.Errorf("unsupported auth mode: %s (supported: 'jwt', 'tls', 'oidc', 'insecure', 'none', or empty for auto-detect)", o.config.AuthMode)
		}
	}
}

// setupAutoDetectAuth attempts to auto-detect available credentials and falls back to insecure if none found.
// This is used when auth mode is empty, "insecure", or "none".
func (o *options) setupAutoDetectAuth(ctx context.Context) error {
	// For explicit "insecure" or "none" mode, skip auto-detection
	if o.config.AuthMode == "insecure" || o.config.AuthMode == "none" {
		authLogger.Debug("Using insecure connection (explicit mode)", "auth_mode", o.config.AuthMode)
		o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

		return nil
	}

	shouldTryOIDC, err := o.shouldAutoDetectOIDC()
	if err != nil {
		return err
	}

	if shouldTryOIDC {
		authLogger.Debug("Auto-detected OIDC authentication signals; using OIDC auth")

		if err := o.setupOIDCAuth(ctx); err != nil {
			return fmt.Errorf("failed to setup auto-detected OIDC auth: %w", err)
		}

		return nil
	}

	// No secure auth signals found; use insecure connection (local development only)
	authLogger.Debug("No auto-detected credentials found, using insecure connection")

	o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	return nil
}

func (o *options) shouldAutoDetectOIDC() (bool, error) {
	if strings.TrimSpace(o.config.AuthToken) != "" {
		return true, nil
	}

	if strings.TrimSpace(o.config.OIDCIssuer) != "" || strings.TrimSpace(o.config.OIDCClientID) != "" {
		return true, nil
	}

	cache, err := ResolveTokenCacheForIssuer(o.config.OIDCIssuer)
	if err != nil {
		if errors.Is(err, ErrNoCachedIssuer) {
			return false, nil
		}

		return false, err
	}

	tok, err := cache.Load()
	if err != nil {
		return false, fmt.Errorf("failed to read OIDC token cache: %w", err)
	}

	return tok != nil && strings.TrimSpace(tok.AccessToken) != "", nil
}

func (o *options) setupJWTAuth(ctx context.Context) error {
	// Validate SPIFFE socket path is set
	if o.config.SpiffeSocketPath == "" {
		return errors.New("spiffe socket path is required for JWT authentication")
	}

	// Validate JWT audience is set
	if o.config.JWTAudience == "" {
		return errors.New("JWT audience is required for JWT authentication")
	}

	// Create SPIFFE client
	client, err := workloadapi.New(ctx, workloadapi.WithAddr(o.config.SpiffeSocketPath))
	if err != nil {
		return fmt.Errorf("failed to create SPIFFE client: %w", err)
	}

	// Create bundle source for verifying server's TLS certificate (X.509-SVID)
	// Note: Use context.Background() because this source must live for the entire client lifetime,
	// not just the initialization phase. It will be properly closed in client.Close().
	bundleSrc, err := workloadapi.NewBundleSource(context.Background(), workloadapi.WithClient(client)) //nolint:contextcheck
	if err != nil {
		_ = client.Close()

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	// Create JWT source for fetching JWT-SVIDs
	// Note: Use context.Background() because this source must live for the entire client lifetime,
	// not just the initialization phase. It will be properly closed in client.Close().
	jwtSource, err := workloadapi.NewJWTSource(context.Background(), workloadapi.WithClient(client)) //nolint:contextcheck
	if err != nil {
		_ = client.Close()
		_ = bundleSrc.Close()

		return fmt.Errorf("failed to create JWT source: %w", err)
	}

	// Use TLS for transport security (server presents X.509-SVID)
	// Client authenticates with JWT-SVID via PerRPCCredentials
	o.authClient = client
	o.bundleSrc = bundleSrc
	o.jwtSource = jwtSource
	o.authOpts = append(o.authOpts,
		grpc.WithTransportCredentials(
			grpccredentials.TLSClientCredentials(bundleSrc, tlsconfig.AuthorizeAny()),
		),
		grpc.WithPerRPCCredentials(newJWTCredentials(jwtSource, o.config.JWTAudience)),
	)

	return nil
}

func (o *options) setupTlsAuth(_ context.Context) error {
	// Validate TLS config is set
	if o.config.TlsCAFile == "" || o.config.TlsCertFile == "" || o.config.TlsKeyFile == "" {
		return errors.New("TLS CA, cert, and key file paths are required for TLS authentication")
	}

	// Load TLS data for tlsConfig
	caData, err := os.ReadFile(o.config.TlsCAFile)
	if err != nil {
		return fmt.Errorf("failed to read TLS CA file: %w", err)
	}

	certData, err := os.ReadFile(o.config.TlsCertFile)
	if err != nil {
		return fmt.Errorf("failed to read TLS cert file: %w", err)
	}

	keyData, err := os.ReadFile(o.config.TlsKeyFile)
	if err != nil {
		return fmt.Errorf("failed to read TLS key file: %w", err)
	}

	// Create certificate from PEM data
	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return fmt.Errorf("failed to create certificate from TLS data: %w", err)
	}

	// Create CA pool from root CAs
	capool := x509.NewCertPool()
	if !capool.AppendCertsFromPEM(caData) {
		return errors.New("failed to append root CA certificate to CA pool")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            capool,
		InsecureSkipVerify: o.config.TlsSkipVerify, //nolint:gosec
	}

	// Update options
	o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	return nil
}
