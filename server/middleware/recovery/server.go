// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package recovery

import (
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
)

// ServerOptions creates unary and stream recovery interceptors for gRPC server.
// These interceptors catch panics and prevent server crashes.
//
// IMPORTANT: These interceptors MUST be the FIRST (outermost) interceptors in the chain
// to catch panics from all other interceptors and handlers.
//
// Example usage:
//
//	serverOpts := []grpc.ServerOption{}
//	// Recovery FIRST (outermost)
//	serverOpts = append(serverOpts, recovery.ServerOptions()...)
//	// Other interceptors after recovery
//	serverOpts = append(serverOpts, logging.ServerOptions(...)...)
//	serverOpts = append(serverOpts, authn.GetServerOptions()...)
func ServerOptions() []grpc.ServerOption {
	opts := DefaultOptions()

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			grpc_recovery.UnaryServerInterceptor(opts...),
		),
		grpc.ChainStreamInterceptor(
			grpc_recovery.StreamServerInterceptor(opts...),
		),
	}
}
