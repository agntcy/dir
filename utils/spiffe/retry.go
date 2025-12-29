// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package spiffe

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
)

// Default retry configuration constants.
const (
	DefaultMaxRetries     = 10
	DefaultInitialBackoff = 500 * time.Millisecond
	DefaultMaxBackoff     = 10 * time.Second
)

// x509SourceGetter defines the interface for getting X509-SVIDs.
// This interface allows us to work with any X509 source that implements GetX509SVID().
type x509SourceGetter interface {
	GetX509SVID() (*x509svid.SVID, error)
}

// X509SourceWithRetry wraps an x509svid.Source and adds retry logic to GetX509SVID().
// This ensures that retry logic is applied not just during setup, but also during
// TLS handshakes when grpccredentials.MTLSClientCredentials or MTLSServerCredentials
// calls GetX509SVID().
// The wrapper implements both x509svid.Source and io.Closer interfaces.
type X509SourceWithRetry struct {
	src            x509svid.Source // The Source interface (workloadapi.X509Source implements this)
	closer         io.Closer       // The Closer interface for Close()
	maxRetries     int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	logger         *slog.Logger
}

// NewX509SourceWithRetry creates a new X509SourceWithRetry wrapper with configurable retry parameters.
//
// Parameters:
//   - src: The X509 source to wrap (must implement x509svid.Source)
//   - closer: The closer for cleanup (typically the same as src)
//   - logger: Logger instance for retry logic logging
//   - maxRetries: Maximum number of retry attempts (use DefaultMaxRetries for default)
//   - initialBackoff: Initial backoff duration between retries (use DefaultInitialBackoff for default)
//   - maxBackoff: Maximum backoff duration (exponential backoff is capped at this value, use DefaultMaxBackoff for default)
func NewX509SourceWithRetry(
	src x509svid.Source,
	closer io.Closer,
	logger *slog.Logger,
	maxRetries int,
	initialBackoff time.Duration,
	maxBackoff time.Duration,
) *X509SourceWithRetry {
	return &X509SourceWithRetry{
		src:            src,
		closer:         closer,
		maxRetries:     maxRetries,
		initialBackoff: initialBackoff,
		maxBackoff:     maxBackoff,
		logger:         logger,
	}
}

// GetX509SVID implements x509svid.Source interface with retry logic.
// This method is called by grpccredentials.MTLSClientCredentials/MTLSServerCredentials during TLS handshake.
func (w *X509SourceWithRetry) GetX509SVID() (*x509svid.SVID, error) {
	w.logger.Info("X509SourceWithRetry.GetX509SVID() called (likely during TLS handshake)",
		"max_retries", w.maxRetries,
		"initial_backoff", w.initialBackoff,
		"max_backoff", w.maxBackoff)

	svid, err := GetX509SVIDWithRetry(w.src, w.maxRetries, w.initialBackoff, w.maxBackoff, w.logger)
	switch {
	case err != nil:
		w.logger.Error("X509SourceWithRetry.GetX509SVID() failed after retries", "error", err, "max_retries", w.maxRetries)
	case svid == nil:
		w.logger.Warn("X509SourceWithRetry.GetX509SVID() returned nil SVID")
	case svid.ID.IsZero():
		w.logger.Warn("X509SourceWithRetry.GetX509SVID() returned SVID with zero ID (no URI SAN)", "has_certificate", svid != nil)
	default:
		w.logger.Info("X509SourceWithRetry.GetX509SVID() succeeded", "spiffe_id", svid.ID.String(), "has_certificate", svid != nil)
	}

	return svid, err
}

// Close implements io.Closer interface by delegating to the wrapped source.
func (w *X509SourceWithRetry) Close() error {
	if err := w.closer.Close(); err != nil {
		return fmt.Errorf("failed to close X509 source: %w", err)
	}

	return nil
}

// GetX509SVIDWithRetry attempts to get a valid X509-SVID with retry logic.
// This handles timing issues where the SPIRE entry hasn't synced to the agent yet
// (common with CronJobs and other short-lived workloads or pod restarts).
// The agent may return a certificate without a URI SAN (SPIFFE ID) if the entry hasn't synced,
// so we must validate that the certificate actually contains a valid SPIFFE ID.
//
// Parameters:
//   - src: The X509 source to get SVIDs from
//   - maxRetries: Maximum number of retry attempts
//   - initialBackoff: Initial backoff duration between retries
//   - maxBackoff: Maximum backoff duration (exponential backoff is capped at this value)
//   - logger: Logger instance for retry logic logging
func GetX509SVIDWithRetry(
	src x509SourceGetter,
	maxRetries int,
	initialBackoff, maxBackoff time.Duration,
	logger *slog.Logger,
) (*x509svid.SVID, error) {
	var (
		svidErr error
		svid    *x509svid.SVID
	)

	logger.Debug("Starting X509-SVID retry logic", "max_retries", maxRetries, "initial_backoff", initialBackoff, "max_backoff", maxBackoff)

	backoff := initialBackoff

	for attempt := range maxRetries {
		logger.Debug("Attempting to get X509-SVID", "attempt", attempt+1)

		svid, svidErr = src.GetX509SVID()
		switch {
		case svidErr == nil && svid != nil && !svid.ID.IsZero():
			// Valid SVID with SPIFFE ID, proceed
			logger.Debug("SVID obtained", "spiffe_id", svid.ID.String(), "is_zero", svid.ID.IsZero())
			logger.Info("Successfully obtained valid X509-SVID with SPIFFE ID", "spiffe_id", svid.ID.String(), "attempt", attempt+1)

			return svid, nil
		case svidErr == nil && svid != nil:
			// Certificate exists but lacks SPIFFE ID - treat as error and retry
			logger.Debug("SVID obtained", "spiffe_id", svid.ID.String(), "is_zero", svid.ID.IsZero())

			svidErr = errors.New("certificate contains no URI SAN (SPIFFE ID)")
			logger.Warn("SVID obtained but lacks URI SAN, retrying", "attempt", attempt+1, "error", svidErr)
		case svidErr != nil:
			logger.Warn("Failed to get X509-SVID", "attempt", attempt+1, "error", svidErr)
		default:
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
