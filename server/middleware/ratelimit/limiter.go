// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/server/middleware/ratelimit/config"
	"github.com/agntcy/dir/utils/logging"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = logging.Logger("ratelimit")

// Limiter defines the interface for rate limiting operations.
// This interface matches the go-grpc-middleware/v2 Limiter interface,
// allowing this implementation to be used with standard interceptors.
//
// Implementations should be thread-safe and support concurrent access.
type Limiter interface {
	// Limit checks if a request should be rate limited.
	// It extracts client identity and method from context, then applies rate limiting rules.
	// Returns an error with codes.ResourceExhausted if rate limit is exceeded.
	Limit(ctx context.Context) error
}

// ClientLimiter implements per-client rate limiting using token bucket algorithm.
// It maintains separate rate limiters for each unique client (identified by SPIFFE ID),
// with support for global limits (for unauthenticated clients) and per-method overrides.
//
// Thread Safety:
// ClientLimiter is safe for concurrent use by multiple goroutines.
// It uses sync.Map for lock-free reads and atomic operations for limiter creation.
type ClientLimiter struct {
	// limiters stores per-client rate limiters (clientID -> *rate.Limiter)
	// Uses sync.Map for efficient concurrent access without locks
	limiters sync.Map

	// globalLimiter is the fallback rate limiter for unauthenticated clients
	globalLimiter *rate.Limiter

	// config holds the rate limiting configuration
	config *config.Config
}

// NewClientLimiter creates a new ClientLimiter with the given configuration.
// It validates the configuration and initializes the global rate limiter.
func NewClientLimiter(cfg *config.Config) (*ClientLimiter, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rate limit config: %w", err)
	}

	// If rate limiting is disabled, return a limiter with nil global limiter
	// Allow() will always return true in this case
	if !cfg.Enabled {
		logger.Info("Rate limiting is disabled")

		return &ClientLimiter{
			config:        cfg,
			globalLimiter: nil,
		}, nil
	}

	// Create global rate limiter for unauthenticated clients
	var globalLimiter *rate.Limiter
	if cfg.GlobalRPS > 0 {
		globalLimiter = rate.NewLimiter(rate.Limit(cfg.GlobalRPS), cfg.GlobalBurst)
		logger.Info("Global rate limiter initialized",
			"rps", cfg.GlobalRPS,
			"burst", cfg.GlobalBurst,
		)
	}

	logger.Info("Client rate limiter initialized",
		"per_client_rps", cfg.PerClientRPS,
		"per_client_burst", cfg.PerClientBurst,
		"method_overrides", len(cfg.MethodLimits),
	)

	return &ClientLimiter{
		globalLimiter: globalLimiter,
		config:        cfg,
	}, nil
}

// Limit checks if a request should be rate limited.
// It implements the go-grpc-middleware/v2 Limiter interface.
//
// The method extracts client identity and method from context, then applies
// the token bucket algorithm:
// - Returns nil if a token is available (request allowed)
// - Returns codes.ResourceExhausted error if rate limited
//
// The method checks rate limits in the following order:
// 1. If rate limiting is disabled, always allow
// 2. Check for method-specific override
// 3. Check per-client limit (if clientID provided)
// 4. Fall back to global limit (for anonymous/unauthenticated clients).
func (l *ClientLimiter) Limit(ctx context.Context) error {
	// If rate limiting is disabled, always allow
	if !l.config.Enabled {
		return nil
	}

	// Extract client ID from context (SPIFFE ID if authenticated)
	clientID := extractClientID(ctx)

	// Extract method name from context
	method, _ := grpc.Method(ctx)

	// Get the appropriate rate limiter
	limiter := l.getLimiterForRequest(clientID, method)

	// If no limiter is configured (both client and global limiters are nil or zero-rate),
	// allow the request
	if limiter == nil {
		return nil
	}

	// Check if request is allowed by the token bucket
	if !limiter.Allow() {
		logger.Warn("Rate limit exceeded",
			"client_id", clientID,
			"method", method,
		)

		return status.Error(codes.ResourceExhausted, "rate limit exceeded") //nolint:wrapcheck // gRPC status error for client
	}

	return nil
}

// extractClientID extracts the client identifier from the gRPC context.
// It returns the SPIFFE ID string if the client is authenticated via authn middleware,
// or an empty string for unauthenticated clients (which will use global rate limit).
func extractClientID(ctx context.Context) string {
	// Try to extract SPIFFE ID from context (set by authentication middleware)
	if spiffeID, ok := authn.SpiffeIDFromContext(ctx); ok {
		return spiffeID.String()
	}

	// No authentication - return empty string to use global rate limiter
	return ""
}

// getLimiterForRequest returns the appropriate rate limiter for a request.
// It checks in order:
// 1. Method-specific override (if configured)
// 2. Per-client limiter (if clientID provided)
// 3. Global limiter (fallback)
//
// Returns nil if no rate limiter is applicable.
func (l *ClientLimiter) getLimiterForRequest(clientID string, method string) *rate.Limiter {
	// Check for method-specific override first
	if method != "" {
		if methodLimit, exists := l.config.MethodLimits[method]; exists {
			// Create a unique key combining client and method
			key := fmt.Sprintf("%s:%s", clientID, method)

			return l.getOrCreateLimiter(key, methodLimit.RPS, methodLimit.Burst)
		}
	}

	// If client ID is provided, use per-client limiter
	if clientID != "" && l.config.PerClientRPS > 0 {
		return l.getOrCreateLimiter(clientID, l.config.PerClientRPS, l.config.PerClientBurst)
	}

	// Fall back to global limiter
	return l.globalLimiter
}

// getOrCreateLimiter gets an existing rate limiter or creates a new one.
// This method is thread-safe and uses sync.Map for efficient concurrent access.
//
// The rate limiter is stored in the limiters map using the provided key.
// If a limiter already exists for the key, it is reused.
// Otherwise, a new limiter is created with the specified rate and burst parameters.
func (l *ClientLimiter) getOrCreateLimiter(key string, rps float64, burst int) *rate.Limiter {
	// Fast path: check if limiter already exists
	if value, exists := l.limiters.Load(key); exists {
		limiter, ok := value.(*rate.Limiter)
		if !ok {
			// This should never happen as we control what goes into the map
			panic(fmt.Sprintf("invalid type in limiters map: expected *rate.Limiter, got %T", value))
		}

		return limiter
	}

	// If RPS is zero, don't create a limiter (unlimited)
	if rps == 0 {
		return nil
	}

	// Slow path: create new limiter
	// Use LoadOrStore to handle race conditions (multiple goroutines creating for same key)
	newLimiter := rate.NewLimiter(rate.Limit(rps), burst)
	actual, loaded := l.limiters.LoadOrStore(key, newLimiter)

	if !loaded {
		logger.Debug("Created new rate limiter",
			"key", key,
			"rps", rps,
			"burst", burst,
		)
	}

	limiter, ok := actual.(*rate.Limiter)
	if !ok {
		// This should never happen as we control what goes into the map
		panic(fmt.Sprintf("invalid type in limiters map: expected *rate.Limiter, got %T", actual))
	}

	return limiter
}

// GetLimiterCount returns the number of active rate limiters.
// This is primarily useful for testing and monitoring.
func (l *ClientLimiter) GetLimiterCount() int {
	count := 0

	l.limiters.Range(func(key, value interface{}) bool {
		count++

		return true
	})

	return count
}
