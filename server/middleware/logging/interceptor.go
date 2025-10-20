// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package logging provides gRPC interceptors for structured request/response logging.
package logging

import (
	"context"
	"log/slog"

	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

// InterceptorLogger adapts slog.Logger to the grpc-middleware Logger interface.
// This allows go-grpc-middleware to use our existing slog-based logging infrastructure.
func InterceptorLogger(l *slog.Logger) grpc_logging.Logger {
	return grpc_logging.LoggerFunc(func(ctx context.Context, lvl grpc_logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}
