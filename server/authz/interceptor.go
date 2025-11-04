// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/server/healthcheck"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InterceptorFn func(ctx context.Context, apiMethod string) error

// NewInterceptor returns a gRPC interceptor that performs authorization checks.
// It expects the SPIFFE ID to already be in the context (set by the authn interceptor).
//
//nolint:wrapcheck
func NewInterceptor(authorizer *Authorizer) InterceptorFn {
	return func(ctx context.Context, apiMethod string) error {
		// Get SPIFFE ID from context (set by authentication interceptor)
		sid, ok := authn.SpiffeIDFromContext(ctx)
		if !ok {
			logger.Error("Authorization failed: no SPIFFE ID in context", "method", apiMethod)

			return status.Error(codes.Unauthenticated, "not authenticated")
		}

		trustDomain := sid.TrustDomain().String()

		// Perform authorization check
		allowed, err := authorizer.Authorize(trustDomain, apiMethod)
		if err != nil {
			logger.Error("Authorization error",
				"error", err,
				"method", apiMethod,
				"trust_domain", trustDomain,
				"spiffe_id", sid.String(),
			)

			return status.Error(codes.Internal, fmt.Sprintf("something went wrong: %v", err))
		}

		if !allowed {
			logger.Warn("Authorization denied",
				"method", apiMethod,
				"trust_domain", trustDomain,
				"spiffe_id", sid.String(),
			)

			return status.Error(codes.PermissionDenied, "not allowed to access "+apiMethod)
		}

		logger.Debug("Authorization successful",
			"method", apiMethod,
			"trust_domain", trustDomain,
			"spiffe_id", sid.String(),
		)

		return nil
	}
}

func UnaryInterceptorFor(fn InterceptorFn) func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
	return func(ctx context.Context, req any, sInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Skip authorization for health check endpoints
		if healthcheck.IsHealthCheckEndpoint(sInfo.FullMethod) {
			return handler(ctx, req)
		}

		if err := fn(ctx, sInfo.FullMethod); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func StreamInterceptorFor(fn InterceptorFn) func(any, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
	return func(srv any, ss grpc.ServerStream, sInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authorization for health check endpoints
		if healthcheck.IsHealthCheckEndpoint(sInfo.FullMethod) {
			return handler(srv, ss)
		}

		if err := fn(ss.Context(), sInfo.FullMethod); err != nil {
			return err
		}

		return handler(srv, ss)
	}
}
