// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/authn/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
)

var logger = logging.Logger("authn")

// Service manages authentication using SPIFFE (X.509 or JWT).
type Service struct {
	mode      config.AuthMode
	audiences []string
	client    *workloadapi.Client
	jwtSource *workloadapi.JWTSource
	x509Src   *workloadapi.X509Source
	bundleSrc *workloadapi.BundleSource
}

// New creates a new authentication service (JWT or X.509 based on config).
func New(ctx context.Context, cfg config.Config) (*Service, error) {
	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authn config: %w", err)
	}

	// Create a client for SPIRE Workload API
	client, err := workloadapi.New(ctx, workloadapi.WithAddr(cfg.SocketPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create workload API client: %w", err)
	}

	service := &Service{
		mode:   cfg.Mode,
		client: client,
	}

	// Initialize based on authentication mode
	switch cfg.Mode {
	case config.AuthModeJWT:
		if err := service.initJWT(ctx, cfg); err != nil {
			_ = client.Close()

			return nil, err
		}

		logger.Info("JWT authentication service initialized", "audiences", cfg.Audiences)

	case config.AuthModeX509:
		if err := service.initX509(ctx); err != nil {
			_ = client.Close()

			return nil, err
		}

		logger.Info("X.509 authentication service initialized")

	default:
		_ = client.Close()

		return nil, fmt.Errorf("unsupported auth mode: %s", cfg.Mode)
	}

	return service, nil
}

// initJWT initializes JWT authentication components.
// For JWT mode, the server presents its X.509-SVID via TLS (for server authentication and encryption),
// while clients authenticate using JWT-SVIDs. This follows the official SPIFFE JWT pattern.
func (s *Service) initJWT(ctx context.Context, cfg config.Config) error {
	// Create X.509 source for server's TLS certificate
	x509Src, err := workloadapi.NewX509Source(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		return fmt.Errorf("failed to create X509 source: %w", err)
	}

	logger.Debug("Created X509 source for JWT mode, waiting for valid SVID")

	// Wait for X509-SVID to be available with retry logic
	// This ensures the server presents a certificate with URI SAN during TLS handshake
	// Critical: If the server starts without a valid SPIFFE ID, clients will reject the connection
	const (
		maxRetries     = 10
		initialBackoff = 500 * time.Millisecond
		maxBackoff     = 10 * time.Second
	)

	svid, svidErr := getX509SVIDWithRetry(x509Src, maxRetries, initialBackoff, maxBackoff)
	if svidErr != nil {
		_ = x509Src.Close()
		logger.Error("Failed to get valid X509-SVID for server after retries", "error", svidErr, "max_retries", maxRetries)
		return fmt.Errorf("failed to get valid X509-SVID for server (SPIRE entry may not be synced yet): %w", svidErr)
	}

	logger.Info("Successfully obtained valid X509-SVID for server", "spiffe_id", svid.ID.String())

	// Create bundle source for trust verification
	bundleSrc, err := workloadapi.NewBundleSource(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		_ = x509Src.Close()

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	// Create JWT source for validating client JWT-SVIDs
	jwtSource, err := workloadapi.NewJWTSource(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		_ = x509Src.Close()
		_ = bundleSrc.Close()

		return fmt.Errorf("failed to create JWT source: %w", err)
	}

	s.x509Src = x509Src
	s.bundleSrc = bundleSrc
	s.jwtSource = jwtSource
	s.audiences = cfg.Audiences

	return nil
}

// initX509 initializes X.509 authentication components.
func (s *Service) initX509(ctx context.Context) error {
	// Create a new X509 source which periodically refetches X509-SVIDs and X.509 bundles
	x509Src, err := workloadapi.NewX509Source(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		return fmt.Errorf("failed to create X509 source: %w", err)
	}

	logger.Debug("Created X509 source for X509 mode, waiting for valid SVID")

	// Wait for X509-SVID to be available with retry logic
	// This ensures the server presents a certificate with URI SAN during TLS handshake
	// Critical: If the server starts without a valid SPIFFE ID, clients will reject the connection
	// with "certificate contains no URI SAN" error
	const (
		maxRetries     = 10
		initialBackoff = 500 * time.Millisecond
		maxBackoff     = 10 * time.Second
	)

	svid, svidErr := getX509SVIDWithRetry(x509Src, maxRetries, initialBackoff, maxBackoff)
	if svidErr != nil {
		_ = x509Src.Close()
		logger.Error("Failed to get valid X509-SVID for server after retries", "error", svidErr, "max_retries", maxRetries)
		return fmt.Errorf("failed to get valid X509-SVID for server (SPIRE entry may not be synced yet): %w", svidErr)
	}

	logger.Info("Successfully obtained valid X509-SVID for server", "spiffe_id", svid.ID.String())

	// Create a new Bundle source which periodically refetches SPIFFE bundles.
	// Required when running Federation.
	bundleSrc, err := workloadapi.NewBundleSource(ctx, workloadapi.WithClient(s.client))
	if err != nil {
		_ = x509Src.Close()

		return fmt.Errorf("failed to create bundle source: %w", err)
	}

	s.x509Src = x509Src
	s.bundleSrc = bundleSrc

	return nil
}

// GetServerOptions returns gRPC server options for authentication.
func (s *Service) GetServerOptions() []grpc.ServerOption {
	switch s.mode {
	case config.AuthModeJWT:
		// JWT mode: Server presents X.509-SVID via TLS, clients authenticate with JWT-SVID
		return []grpc.ServerOption{
			grpc.Creds(
				grpccredentials.TLSServerCredentials(s.x509Src),
			),
			grpc.ChainUnaryInterceptor(JWTUnaryInterceptor(s.jwtSource, s.audiences)),
			grpc.ChainStreamInterceptor(JWTStreamInterceptor(s.jwtSource, s.audiences)),
		}

	case config.AuthModeX509:
		return []grpc.ServerOption{
			grpc.Creds(
				grpccredentials.MTLSServerCredentials(s.x509Src, s.bundleSrc, tlsconfig.AuthorizeAny()),
			),
			grpc.ChainUnaryInterceptor(X509UnaryInterceptor()),
			grpc.ChainStreamInterceptor(X509StreamInterceptor()),
		}

	default:
		logger.Error("Unsupported auth mode", "mode", s.mode)

		return []grpc.ServerOption{}
	}
}

// Stop closes the workload API client and all sources.
//
//nolint:wrapcheck
func (s *Service) Stop() error {
	if s.jwtSource != nil {
		if err := s.jwtSource.Close(); err != nil {
			logger.Error("Failed to close JWT source", "error", err)
		}
	}

	if s.x509Src != nil {
		if err := s.x509Src.Close(); err != nil {
			logger.Error("Failed to close X509 source", "error", err)
		}
	}

	if s.bundleSrc != nil {
		if err := s.bundleSrc.Close(); err != nil {
			logger.Error("Failed to close bundle source", "error", err)
		}
	}

	return s.client.Close()
}

// x509SourceGetter defines the interface for getting X509-SVIDs.
type x509SourceGetter interface {
	GetX509SVID() (*x509svid.SVID, error)
}

// getX509SVIDWithRetry attempts to get a valid X509-SVID with retry logic.
// This handles timing issues where the SPIRE entry hasn't synced to the agent yet
// (common with short-lived workloads or pod restarts).
// The agent may return a certificate without a URI SAN (SPIFFE ID) if the entry hasn't synced,
// so we must validate that the certificate actually contains a valid SPIFFE ID.
//
// Parameters:
//   - src: The X509 source to get SVIDs from
//   - maxRetries: Maximum number of retry attempts
//   - initialBackoff: Initial backoff duration between retries
//   - maxBackoff: Maximum backoff duration (exponential backoff is capped at this value)
func getX509SVIDWithRetry(src x509SourceGetter, maxRetries int, initialBackoff, maxBackoff time.Duration) (*x509svid.SVID, error) {
	var (
		svidErr error
		svid    *x509svid.SVID
	)

	logger.Debug("Starting X509-SVID retry logic", "max_retries", maxRetries, "initial_backoff", initialBackoff, "max_backoff", maxBackoff)

	backoff := initialBackoff

	for attempt := range maxRetries {
		logger.Debug("Attempting to get X509-SVID", "attempt", attempt+1)
		svid, svidErr = src.GetX509SVID()
		if svidErr == nil && svid != nil {
			logger.Debug("SVID obtained", "spiffe_id", svid.ID.String(), "is_zero", svid.ID.IsZero())
			// Validate that the SVID has a valid SPIFFE ID (URI SAN)
			// The agent may return a certificate without a URI SAN if the entry hasn't synced yet
			if !svid.ID.IsZero() {
				// Valid SVID with SPIFFE ID, proceed
				logger.Info("Successfully obtained valid X509-SVID with SPIFFE ID", "spiffe_id", svid.ID.String(), "attempt", attempt+1)
				return svid, nil
			}
			// Certificate exists but lacks SPIFFE ID - treat as error and retry
			svidErr = errors.New("certificate contains no URI SAN (SPIFFE ID)")
			logger.Warn("SVID obtained but lacks URI SAN, retrying", "attempt", attempt+1, "error", svidErr)
		} else if svidErr != nil {
			logger.Warn("Failed to get X509-SVID", "attempt", attempt+1, "error", svidErr)
		} else {
			logger.Warn("GetX509SVID returned nil SVID with no error, retrying", "attempt", attempt+1)
			svidErr = errors.New("nil SVID returned") // Force retry
		}

		if attempt < maxRetries-1 {
			logger.Debug("Backing off before next retry", "duration", backoff, "attempt", attempt+1)
			// Exponential backoff: initialBackoff, initialBackoff*2, initialBackoff*4, ... (capped at maxBackoff)
			time.Sleep(backoff)

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	if svidErr == nil {
		svidErr = errors.New("certificate contains no URI SAN (SPIFFE ID)")
	}

	logger.Error("Failed to get valid X509-SVID after retries", "max_retries", maxRetries, "error", svidErr, "final_svid", svid)
	return nil, fmt.Errorf("failed to get valid X509-SVID after %d retries: %w", maxRetries, svidErr)
}
