// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
)

// ServerOptions returns gRPC server options for Prometheus metrics collection.
// Interceptors are chained and collect metrics for all gRPC methods automatically.
func ServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.ChainStreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	}
}

// InitializeMetrics registers gRPC metrics with the Prometheus registry.
// Must be called after all gRPC services are registered on the server.
func InitializeMetrics(grpcServer *grpc.Server, metricsServer *Server) {
	// Initialize gRPC metrics with all registered services
	grpc_prometheus.Register(grpcServer)

	// Register grpc_prometheus metrics with our custom registry
	metricsServer.Registry().MustRegister(grpc_prometheus.DefaultServerMetrics)
}
