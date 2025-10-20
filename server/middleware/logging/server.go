// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"log/slog"

	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

// ServerOptions creates unary and stream interceptors for gRPC server logging.
// If verbose is true, uses VerboseOptions (includes payloads), otherwise uses DefaultOptions.
func ServerOptions(logger *slog.Logger, verbose bool) []grpc.ServerOption {
	// Create the interceptor logger adapter
	interceptorLogger := InterceptorLogger(logger)

	// Choose options based on verbose mode
	var opts []grpc_logging.Option
	if verbose {
		opts = VerboseOptions()
	} else {
		opts = DefaultOptions()
	}

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			grpc_logging.UnaryServerInterceptor(interceptorLogger, opts...),
		),
		grpc.ChainStreamInterceptor(
			grpc_logging.StreamServerInterceptor(interceptorLogger, opts...),
		),
	}
}
