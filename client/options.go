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

	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/dir/utils/spiffe"
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
		case "x509":
			// NOTE: x509 source must live for the entire client lifetime, not just the initialization phase
			return o.setupX509Auth(ctx) //nolint:contextcheck
		case "token":
			return o.setupSpiffeAuth(ctx)
		case "tls":
			return o.setupTlsAuth(ctx)
		case "github":
			// Explicit GitHub auth mode - use cached credentials or fail
			return o.setupGitHubAuth(ctx)
		case "insecure", "none", "":
			// Insecure/none/empty auth mode - try auto-detection first, fallback to insecure
			return o.setupAutoDetectAuth(ctx)
		default:
			// Invalid auth mode specified - return error to prevent silent security issues
			return fmt.Errorf("unsupported auth mode: %s (supported: 'jwt', 'x509', 'token', 'tls', 'github', 'insecure', 'none', or empty for auto-detect)", o.config.AuthMode)
		}
	}
}

// setupAutoDetectAuth attempts to auto-detect available credentials and falls back to insecure if none found.
// This is used when auth mode is empty, "insecure", or "none".
func (o *options) setupAutoDetectAuth(_ context.Context) error {
	// For explicit "insecure" or "none" mode, skip auto-detection
	if o.config.AuthMode == "insecure" || o.config.AuthMode == "none" {
		authLogger.Debug("Using insecure connection (explicit mode)", "auth_mode", o.config.AuthMode)
		o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

		return nil
	}

	// Empty auth mode - auto-detect based on available credentials
	var token string

	// 1. Check if token is provided via config/flag/env
	if o.config.GitHubToken != "" {
		authLogger.Debug("Auto-detected token from config/environment")

		token = o.config.GitHubToken
	} else {
		// 2. Check for cached GitHub OAuth token
		cache := NewTokenCache()

		cachedToken, err := cache.GetValidToken()
		if err != nil {
			authLogger.Debug("Error loading cached GitHub token, falling back to insecure", "error", err)
		}

		if cachedToken != nil {
			authLogger.Debug("Auto-detected cached GitHub OAuth token", "user", cachedToken.User)
			token = cachedToken.AccessToken
		}
	}

	// If token found (either from config or cache), use it
	if token != "" {
		// Use TLS for external Envoy gateway (ingress expects HTTPS on port 443)
		// System CA pool for validating ingress TLS certificates
		tlsConfig := &tls.Config{
			InsecureSkipVerify: o.config.TlsSkipVerify, //nolint:gosec
		}
		o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

		// Add GitHub token as Bearer token in Authorization header
		o.authOpts = append(o.authOpts, grpc.WithPerRPCCredentials(newGitHubCredentials(token)))

		authLogger.Debug("GitHub authentication configured via auto-detect")

		return nil
	}

	// No cached credentials - use insecure connection (for local development only)
	authLogger.Debug("No cached credentials found, using insecure connection")

	o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	return nil
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

	authLogger.Debug("Setting up X509 authentication", "spiffe_socket_path", o.config.SpiffeSocketPath)

	// Create SPIFFE client
	client, err := workloadapi.New(ctx, workloadapi.WithAddr(o.config.SpiffeSocketPath))
	if err != nil {
		return fmt.Errorf("failed to create SPIFFE client: %w", err)
	}

	authLogger.Debug("Created SPIFFE workload API client")

	// Create SPIFFE x509 services
	// Note: Use context.Background() because this source must live for the entire client lifetime,
	// not just the initialization phase. It will be properly closed in client.Close().
	x509Src, err := workloadapi.NewX509Source(context.Background(), workloadapi.WithClient(client)) //nolint:contextcheck
	if err != nil {
		_ = client.Close()

		return fmt.Errorf("failed to create x509 source: %w", err)
	}

	authLogger.Debug("Created X509 source, starting retry logic to get valid SVID")

	// Wait for X509-SVID to be available with retry logic
	// This handles timing issues where the SPIRE entry hasn't been synced to the agent yet
	// (common with CronJobs and other short-lived workloads)
	// The agent may return a certificate without a URI SAN (SPIFFE ID) if the entry hasn't synced,
	// so we must validate that the certificate actually contains a valid SPIFFE ID.
	svid, svidErr := spiffe.GetX509SVIDWithRetry(
		x509Src,
		spiffe.DefaultMaxRetries,
		spiffe.DefaultInitialBackoff,
		spiffe.DefaultMaxBackoff,
		authLogger,
	)
	if svidErr != nil {
		_ = client.Close()
		_ = x509Src.Close()

		authLogger.Error("Failed to get valid X509-SVID after retries", "error", svidErr, "max_retries", spiffe.DefaultMaxRetries)

		return fmt.Errorf("failed to get valid X509-SVID after retries (SPIRE entry may not be synced yet): %w", svidErr)
	}

	authLogger.Info("Successfully obtained valid X509-SVID", "spiffe_id", svid.ID.String())

	// Wrap x509Src with retry logic so GetX509SVID() calls during TLS handshake also retry
	// This is critical because grpccredentials.MTLSClientCredentials calls GetX509SVID()
	// during the actual TLS handshake, not just during setup. Without this wrapper,
	// the TLS handshake may fail if the certificate doesn't have a URI SAN at that moment.
	//
	// Connection flow: dirctl → Ingress (TLS passthrough) → apiserver pod
	// The TLS handshake happens between dirctl and apiserver, and during this handshake,
	// grpccredentials.MTLSClientCredentials calls GetX509SVID() again.
	//
	// Note: x509Src is *workloadapi.X509Source (concrete type that implements x509svid.Source).
	// We use it directly as the Source interface and also as io.Closer.
	wrappedX509Src := spiffe.NewX509SourceWithRetry(
		x509Src, // Use pointer directly (implements x509svid.Source)
		x509Src, // Same pointer (implements io.Closer)
		authLogger,
		spiffe.DefaultMaxRetries,
		spiffe.DefaultInitialBackoff,
		spiffe.DefaultMaxBackoff,
	)

	authLogger.Debug("Created X509SourceWithRetry wrapper",
		"wrapped_type", fmt.Sprintf("%T", wrappedX509Src),
		"src_type", fmt.Sprintf("%T", x509Src),
		"implements_source", true)

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
	o.x509Src = wrappedX509Src // Store wrapped source for cleanup
	o.bundleSrc = bundleSrc

	authLogger.Debug("Creating MTLSClientCredentials with wrapped source",
		"wrapped_source_type", fmt.Sprintf("%T", wrappedX509Src),
		"wrapped_implements_source", true)

	creds := grpccredentials.MTLSClientCredentials(wrappedX509Src, bundleSrc, tlsconfig.AuthorizeAny())
	o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(creds))

	authLogger.Debug("MTLSClientCredentials created successfully, wrapper will be used for TLS handshake")

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
	//nolint:gosec // G117: intentional private key field
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

func (o *options) setupGitHubAuth(_ context.Context) error {
	authLogger.Debug("Setting up GitHub authentication")

	var token string

	// 1. First, check if token is provided via config/flag/env
	//    This allows CI/CD to use PATs: export DIRECTORY_CLIENT_GITHUB_TOKEN=ghp_xxx
	if o.config.GitHubToken != "" {
		authLogger.Debug("Using token from config/environment")

		token = o.config.GitHubToken
	} else {
		// 2. Fall back to cached OAuth token from interactive login
		cache := NewTokenCache()

		cachedToken, err := cache.GetValidToken()
		if err != nil {
			authLogger.Debug("Error loading cached token", "error", err)
		}

		if cachedToken == nil {
			return errors.New("not authenticated with GitHub. Run 'dirctl auth login' or set DIRECTORY_CLIENT_GITHUB_TOKEN environment variable")
		}

		authLogger.Debug("Using cached GitHub OAuth token", "user", cachedToken.User)
		token = cachedToken.AccessToken
	}

	// Use TLS for external Envoy gateway (ingress expects HTTPS on port 443)
	// System CA pool for validating ingress TLS certificates
	tlsConfig := &tls.Config{
		InsecureSkipVerify: o.config.TlsSkipVerify, //nolint:gosec
	}
	o.authOpts = append(o.authOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	// Add GitHub token as Bearer token in Authorization header
	o.authOpts = append(o.authOpts, grpc.WithPerRPCCredentials(newGitHubCredentials(token)))

	authLogger.Debug("GitHub authentication configured")

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
