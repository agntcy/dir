// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"context"

	"github.com/agntcy/dir/server/authn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor that performs rate limiting.
// It extracts the client identity from the context (SPIFFE ID if authenticated),
// checks the rate limit using the provided ClientLimiter, and returns ResourceExhausted
// error if the limit is exceeded.
func UnaryServerInterceptor(limiter *ClientLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Extract client ID from context (SPIFFE ID if authenticated, empty if not)
		clientID := extractClientID(ctx)

		// Check rate limit
		if !limiter.Allow(ctx, clientID, info.FullMethod) {
			logger.Warn("Rate limit exceeded",
				"client_id", clientID,
				"method", info.FullMethod,
			)

			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		// Rate limit passed, proceed with the request
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor that performs rate limiting.
// It extracts the client identity from the context (SPIFFE ID if authenticated),
// checks the rate limit using the provided ClientLimiter, and returns ResourceExhausted
// error if the limit is exceeded.
//
// Note: Rate limiting is applied when the stream is initiated, not per message.
// This is the standard approach for stream rate limiting to avoid overhead.
func StreamServerInterceptor(limiter *ClientLimiter) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// Extract client ID from context (SPIFFE ID if authenticated, empty if not)
		clientID := extractClientID(ctx)

		// Check rate limit
		if !limiter.Allow(ctx, clientID, info.FullMethod) {
			logger.Warn("Rate limit exceeded",
				"client_id", clientID,
				"method", info.FullMethod,
			)

			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		// Rate limit passed, proceed with the stream
		return handler(srv, ss)
	}
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
