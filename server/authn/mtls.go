// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewMTLSInterceptor returns a gRPC interceptor that extracts the SPIFFE ID from mTLS peer
// and adds it to the context for downstream authorization checks.
func NewMTLSInterceptor() MTLSInterceptorFn {
	return func(ctx context.Context) (context.Context, error) {
		// Extract SPIFFE ID from mTLS peer certificate
		sid, ok := grpccredentials.PeerIDFromContext(ctx)
		if !ok {
			logger.Error("mTLS authentication failed: no peer ID in context")

			return ctx, status.Error(codes.Unauthenticated, "not authenticated via mTLS")
		}

		logger.Debug("mTLS authentication successful",
			"spiffe_id", sid.String(),
			"trust_domain", sid.TrustDomain().String(),
		)

		// Store the SPIFFE ID in context using the same approach as JWT
		return context.WithValue(ctx, SpiffeIDContextKey, sid), nil
	}
}

// MTLSInterceptorFn is a function that performs mTLS authentication.
type MTLSInterceptorFn func(ctx context.Context) (context.Context, error)

// mtlsUnaryInterceptorFor wraps the mTLS interceptor function for unary RPCs.
func mtlsUnaryInterceptorFor(fn MTLSInterceptorFn) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		newCtx, err := fn(ctx)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// mtlsStreamInterceptorFor wraps the mTLS interceptor function for stream RPCs.
func mtlsStreamInterceptorFor(fn MTLSInterceptorFn) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
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

// MTLSUnaryInterceptor is a convenience wrapper for mTLS unary authentication.
func MTLSUnaryInterceptor() grpc.UnaryServerInterceptor {
	return mtlsUnaryInterceptorFor(NewMTLSInterceptor())
}

// MTLSStreamInterceptor is a convenience wrapper for mTLS stream authentication.
func MTLSStreamInterceptor() grpc.StreamServerInterceptor {
	return mtlsStreamInterceptorFor(NewMTLSInterceptor())
}
