// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

		// Setup authentication based on AuthMode
		switch o.config.AuthMode {
		case "jwt":
			// NOTE: jwt source must live for the entire client lifetime, not just the initialization phase
			return o.setupJWTAuth(ctx) //nolint:contextcheck
		case "x509":
			// NOTE: x509 source must live for the entire client lifetime, not just the initialization phase
			return o.setupX509Auth(ctx) //nolint:contextcheck
		case "token":
			return o.setupSpiffeAuth(ctx)
		case "tls":
			return o.setupTlsAuth(ctx)
		case "":
			// Empty auth mode - use insecure connection (for development/testing only)
			o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

			return nil
		default:
			// Invalid auth mode specified - return error to prevent silent security issues
			return fmt.Errorf("unsupported auth mode: %s (supported: 'jwt', 'x509', 'token', or empty for insecure)", o.config.AuthMode)
		}
	}
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

func (o *options) setupX509Auth(ctx context.Context) error {
	// Validate SPIFFE socket path is set
	if o.config.SpiffeSocketPath == "" {
		return errors.New("spiffe socket path is required for x509 authentication")
	}

	// Create SPIFFE client
	client, err := workloadapi.New(ctx, workloadapi.WithAddr(o.config.SpiffeSocketPath))
	if err != nil {
		return fmt.Errorf("failed to create SPIFFE client: %w", err)
	}

	// Create SPIFFE x509 services
	// Note: Use context.Background() because this source must live for the entire client lifetime,
	// not just the initialization phase. It will be properly closed in client.Close().
	x509Src, err := workloadapi.NewX509Source(context.Background(), workloadapi.WithClient(client)) //nolint:contextcheck
	if err != nil {
		_ = client.Close()

		return fmt.Errorf("failed to create x509 source: %w", err)
	}

	// Create SPIFFE bundle services
	// Note: Use context.Background() because this source must live for the entire client lifetime,
	// not just the initialization phase. It will be properly closed in client.Close().
	bundleSrc, err := workloadapi.NewBundleSource(context.Background(), workloadapi.WithClient(client)) //nolint:contextcheck
	if err != nil {
		_ = client.Close()
		_ = x509Src.Close() // Fix Issue #4: Close x509Src on error

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	// Update options
	o.authClient = client
	o.x509Src = x509Src
	o.bundleSrc = bundleSrc
	o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(
		grpccredentials.MTLSClientCredentials(x509Src, bundleSrc, tlsconfig.AuthorizeAny()),
	))

	return nil
}

func (o *options) setupSpiffeAuth(_ context.Context) error {
	// Validate token file is set
	if o.config.SpiffeToken == "" {
		return errors.New("spiffe token file path is required for token authentication")
	}

	// Read token file
	tokenData, err := os.ReadFile(o.config.SpiffeToken)
	if err != nil {
		return fmt.Errorf("failed to read SPIFFE token file: %w", err)
	}

	// SpiffeTokenData represents the structure of SPIFFE token JSON
	type SpiffeTokenData struct {
		X509SVID   []string `json:"x509_svid"`   // DER-encoded certificates in base64
		PrivateKey string   `json:"private_key"` // DER-encoded private key in base64
		RootCAs    []string `json:"root_cas"`    // DER-encoded root CA certificates in base64
	}

	// Parse SPIFFE token JSON
	var spiffeData []SpiffeTokenData
	if err := json.Unmarshal(tokenData, &spiffeData); err != nil {
		return fmt.Errorf("failed to parse SPIFFE token: %w", err)
	}

	if len(spiffeData) == 0 {
		return errors.New("no SPIFFE data found in token")
	}

	// Use the first SPIFFE data entry
	data := spiffeData[0]

	// Parse the certificate chain
	if len(data.X509SVID) == 0 {
		return errors.New("no X.509 SVID certificates found")
	}

	// From base64 DER to PEM
	certDER, err := base64.StdEncoding.DecodeString(data.X509SVID[0])
	if err != nil {
		return fmt.Errorf("failed to decode certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// The private key is base64-encoded DER format
	keyDER, err := base64.StdEncoding.DecodeString(data.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to decode private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	})

	// Create certificate from PEM data
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("failed to create certificate from SPIFFE data: %w", err)
	}

	// Create CA pool from root CAs
	capool := x509.NewCertPool()

	for _, rootCA := range data.RootCAs {
		// Root CAs are also base64-encoded DER
		caDER, err := base64.StdEncoding.DecodeString(rootCA)
		if err != nil {
			return fmt.Errorf("failed to decode root CA: %w", err)
		}

		caPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caDER,
		})

		if !capool.AppendCertsFromPEM(caPEM) {
			return errors.New("failed to append root CA certificate to CA pool")
		}
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
