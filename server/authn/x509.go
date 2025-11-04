// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"

	"github.com/agntcy/dir/server/healthcheck"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewX509Interceptor returns a gRPC interceptor that extracts the SPIFFE ID from X.509 peer
// and adds it to the context for downstream authorization checks.
func NewX509Interceptor() X509InterceptorFn {
	return func(ctx context.Context) (context.Context, error) {
		// Extract SPIFFE ID from X.509 peer certificate
		sid, ok := grpccredentials.PeerIDFromContext(ctx)
		if !ok {
			logger.Error("X.509 authentication failed: no peer ID in context")

			return ctx, status.Error(codes.Unauthenticated, "not authenticated via X.509")
		}

		logger.Debug("X.509 authentication successful",
			"spiffe_id", sid.String(),
			"trust_domain", sid.TrustDomain().String(),
		)

		// Store the SPIFFE ID in context using the same approach as JWT
		return context.WithValue(ctx, SpiffeIDContextKey, sid), nil
	}
}

// X509InterceptorFn is a function that performs X.509 authentication.
type X509InterceptorFn func(ctx context.Context) (context.Context, error)

// x509UnaryInterceptorFor wraps the X.509 interceptor function for unary RPCs.
func x509UnaryInterceptorFor(fn X509InterceptorFn) grpc.UnaryServerInterceptor {
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

// x509StreamInterceptorFor wraps the X.509 interceptor function for stream RPCs.
func x509StreamInterceptorFor(fn X509InterceptorFn) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication for health check endpoints
		if healthcheck.IsHealthCheckEndpoint(info.FullMethod) {
			return handler(srv, ss)
		}

		newCtx, err := fn(ss.Context())
		if err != nil {
			return err
		}

		// Create a wrapped stream with the new context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// X509UnaryInterceptor is a convenience wrapper for X.509 unary authentication.
func X509UnaryInterceptor() grpc.UnaryServerInterceptor {
	return x509UnaryInterceptorFor(NewX509Interceptor())
}

// X509StreamInterceptor is a convenience wrapper for X.509 stream authentication.
func X509StreamInterceptor() grpc.StreamServerInterceptor {
	return x509StreamInterceptorFor(NewX509Interceptor())
}
