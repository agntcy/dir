// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"github.com/agntcy/dir/server/middleware/ratelimit/config"
	"google.golang.org/grpc"
)

// ServerOptions creates unary and stream rate limiting interceptors for gRPC server.
// These interceptors enforce rate limits based on client identity (SPIFFE ID) and method.
//
// IMPORTANT: These interceptors should be placed AFTER recovery middleware but BEFORE
// authentication/authorization middleware in the interceptor chain. This ensures:
// 1. Panics are caught by recovery middleware
// 2. Rate limiting protects authentication/authorization processing
// 3. DDoS attacks are mitigated before expensive auth operations
//
// Example usage:
//
//	serverOpts := []grpc.ServerOption{}
//	// Recovery FIRST (outermost)
//	serverOpts = append(serverOpts, recovery.ServerOptions()...)
//	// Rate limiting AFTER recovery
//	if rateLimitCfg.Enabled {
//	    serverOpts = append(serverOpts, ratelimit.ServerOptions(rateLimitCfg)...)
//	}
//	// Logging and auth interceptors after rate limiting
//	serverOpts = append(serverOpts, logging.ServerOptions(...)...)
//	serverOpts = append(serverOpts, authn.GetServerOptions()...)
func ServerOptions(cfg *config.Config) ([]grpc.ServerOption, error) {
	// Create the client limiter
	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		return nil, err
	}

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			UnaryServerInterceptor(limiter),
		),
		grpc.ChainStreamInterceptor(
			StreamServerInterceptor(limiter),
		),
	}, nil
}
