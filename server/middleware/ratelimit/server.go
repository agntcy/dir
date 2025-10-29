// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"github.com/agntcy/dir/server/middleware/ratelimit/config"
	grpc_ratelimit "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/ratelimit"
	"google.golang.org/grpc"
)

// ServerOptions creates unary and stream rate limiting interceptors for gRPC server.
// These interceptors enforce rate limits based on client identity (SPIFFE ID) and method.
//
// This uses the go-grpc-middleware/v2 rate limiting interceptors with a custom
// Limiter implementation that supports per-client and per-method rate limiting.
//
// Returns an error if the configuration is invalid (e.g., negative values).
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
//	    rateLimitOpts, err := ratelimit.ServerOptions(rateLimitCfg)
//	    if err != nil {
//	        return err
//	    }
//	    serverOpts = append(serverOpts, rateLimitOpts...)
//	}
//	// Logging and auth interceptors after rate limiting
//	serverOpts = append(serverOpts, logging.ServerOptions(...)...)
//	serverOpts = append(serverOpts, authn.GetServerOptions()...)
func ServerOptions(cfg *config.Config) ([]grpc.ServerOption, error) {
	// Create the client limiter that implements go-grpc-middleware/v2 Limiter interface
	limiter, err := NewClientLimiter(cfg)
	if err != nil {
		return nil, err
	}

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			grpc_ratelimit.UnaryServerInterceptor(limiter),
		),
		grpc.ChainStreamInterceptor(
			grpc_ratelimit.StreamServerInterceptor(limiter),
		),
	}, nil
}
