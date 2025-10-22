// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package recovery

import (
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
)

// DefaultOptions returns the recommended recovery configuration for production use.
// It uses the custom PanicHandler for comprehensive logging and error handling.
//
// The recovery handler will:
//   - Catch panics from handlers and interceptors
//   - Log full stack traces with context (method, SPIFFE ID)
//   - Return proper gRPC errors (codes.Internal) to clients
//   - Keep the server running after panic recovery
func DefaultOptions() []grpc_recovery.Option {
	return []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandlerContext(PanicHandler),
	}
}
