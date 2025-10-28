// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/agntcy/dir/server/middleware/ratelimit/config"
	"github.com/agntcy/dir/utils/logging"
	"golang.org/x/time/rate"
)

var logger = logging.Logger("ratelimit")

// Limiter defines the interface for rate limiting operations.
// Implementations should be thread-safe and support concurrent access.
type Limiter interface {
	// Allow reports whether an event may happen now for the given client and method.
	// It returns true if the event is allowed, false if rate limited.
	Allow(ctx context.Context, clientID string, method string) bool

	// Wait blocks until an event can happen or the context is cancelled.
	// It returns an error if the context is cancelled before the event can happen.
	Wait(ctx context.Context, clientID string, method string) error
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

// Allow reports whether an event may happen now for the given client and method.
// It implements the token bucket algorithm:
// - Returns true if a token is available (request allowed)
// - Returns false if no tokens available (request rate limited)
//
// The method checks rate limits in the following order:
// 1. If rate limiting is disabled, always allow
// 2. Check for method-specific override
// 3. Check per-client limit (if clientID provided)
// 4. Fall back to global limit (for anonymous/unauthenticated clients).
func (l *ClientLimiter) Allow(ctx context.Context, clientID string, method string) bool {
	// If rate limiting is disabled, always allow
	if !l.config.Enabled {
		return true
	}

	// Get the appropriate rate limiter
	limiter := l.getLimiterForRequest(clientID, method)

	// If no limiter is configured (both client and global limiters are nil or zero-rate),
	// allow the request
	if limiter == nil {
		return true
	}

	// Check if request is allowed by the token bucket
	allowed := limiter.Allow()

	if !allowed {
		logger.Warn("Rate limit exceeded",
			"client_id", clientID,
			"method", method,
		)
	}

	return allowed
}

// Wait blocks until an event can happen or the context is cancelled.
// It waits for a token to become available in the token bucket.
// Returns an error if the context is cancelled before a token is available.
func (l *ClientLimiter) Wait(ctx context.Context, clientID string, method string) error {
	// If rate limiting is disabled, return immediately
	if !l.config.Enabled {
		return nil
	}

	// Get the appropriate rate limiter
	limiter := l.getLimiterForRequest(clientID, method)

	// If no limiter is configured, return immediately
	if limiter == nil {
		return nil
	}

	// Wait for a token to become available
	if err := limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait failed for client %s method %s: %w", clientID, method, err)
	}

	return nil
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
