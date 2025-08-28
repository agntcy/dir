// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InterceptorFn func(ctx context.Context, apiMethod string) error

// Interceptor returns a gRPC interceptor that performs authorization checks.
//
//nolint:wrapcheck
func NewInterceptor(authorizer *Authorizer) InterceptorFn {
	return func(ctx context.Context, apiMethod string) error {
		sid, ok := grpccredentials.PeerIDFromContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "not authenticated")
		}

		trustDomain := sid.TrustDomain().String()

		allowed, err := authorizer.Authorize(trustDomain, apiMethod)
		if err != nil {
			logger.Error("Authorization error",
				"error", err,
				"method", apiMethod,
				"trust_domain", trustDomain,
			)

			return status.Error(codes.Internal, fmt.Sprintf("something went wrong: %v", err))
		}

		if !allowed {
			return status.Error(codes.PermissionDenied, "not allowed to access "+apiMethod)
		}

		return nil
	}
}

func UnaryInterceptorFor(fn InterceptorFn) func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
	return func(ctx context.Context, req any, sInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := fn(ctx, sInfo.FullMethod); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func StreamInterceptorFor(fn InterceptorFn) func(any, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
	return func(srv any, ss grpc.ServerStream, sInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := fn(ss.Context(), sInfo.FullMethod); err != nil {
			return err
		}

		return handler(srv, ss)
	}
}
