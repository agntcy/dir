// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/server/healthcheck"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Context key for storing authenticated SPIFFE ID.
type contextKey string

const (
	// SpiffeIDContextKey is the context key for the authenticated SPIFFE ID.
	SpiffeIDContextKey contextKey = "spiffe-id"
)

// JWTInterceptorFn is a function that performs JWT authentication.
type JWTInterceptorFn func(ctx context.Context) (context.Context, error)

// NewJWTInterceptor returns an interceptor function that validates JWT tokens.
func NewJWTInterceptor(jwtSource *workloadapi.JWTSource, audiences []string) JWTInterceptorFn {
	return func(ctx context.Context) (context.Context, error) {
		// Extract JWT from metadata
		token, err := extractToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, fmt.Sprintf("failed to extract token: %v", err))
		}

		// Validate JWT for each audience until one succeeds
		var (
			svid    *jwtsvid.SVID
			lastErr error
		)

		for _, audience := range audiences {
			svid, lastErr = jwtsvid.ParseAndValidate(token, jwtSource, []string{audience})
			if lastErr == nil {
				break
			}
		}

		if lastErr != nil {
			logger.Warn("JWT validation failed",
				"error", lastErr,
				"audiences", audiences,
			)

			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		// Extract SPIFFE ID
		spiffeID := svid.ID

		logger.Debug("JWT authenticated",
			"spiffe_id", spiffeID.String(),
			"audience", svid.Audience,
		)

		// Store SPIFFE ID in context for downstream handlers
		ctx = context.WithValue(ctx, SpiffeIDContextKey, spiffeID)

		return ctx, nil
	}
}

// extractToken extracts the JWT token from gRPC metadata.
func extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return "", errors.New("missing authorization header")
	}

	// Expected format: "Bearer <token>"
	const expectedParts = 2

	parts := strings.SplitN(authHeader[0], " ", expectedParts)

	if len(parts) != expectedParts || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}

// SpiffeIDFromContext extracts the SPIFFE ID from the context.
func SpiffeIDFromContext(ctx context.Context) (spiffeid.ID, bool) {
	id, ok := ctx.Value(SpiffeIDContextKey).(spiffeid.ID)

	return id, ok
}

// jwtUnaryInterceptorFor wraps the JWT interceptor function for unary RPCs.
func jwtUnaryInterceptorFor(fn JWTInterceptorFn) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Skip authentication for health check endpoints
		if healthcheck.IsHealthCheckEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		newCtx, err := fn(ctx)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// jwtStreamInterceptorFor wraps the JWT interceptor function for stream RPCs.
func jwtStreamInterceptorFor(fn JWTInterceptorFn) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication for health check endpoints
		if healthcheck.IsHealthCheckEndpoint(info.FullMethod) {
			return handler(srv, ss)
		}

		newCtx, err := fn(ss.Context())
		if err != nil {
			return err
		}

		// Wrap the stream to use the new context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// JWTUnaryInterceptor is a convenience wrapper for JWT unary authentication.
func JWTUnaryInterceptor(jwtSource *workloadapi.JWTSource, audiences []string) grpc.UnaryServerInterceptor {
	return jwtUnaryInterceptorFor(NewJWTInterceptor(jwtSource, audiences))
}

// JWTStreamInterceptor is a convenience wrapper for JWT stream authentication.
func JWTStreamInterceptor(jwtSource *workloadapi.JWTSource, audiences []string) grpc.StreamServerInterceptor {
	return jwtStreamInterceptorFor(NewJWTInterceptor(jwtSource, audiences))
}

// wrappedServerStream wraps a grpc.ServerStream to override the context.
//
//nolint:containedctx // Context is required for gRPC stream wrapping
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
