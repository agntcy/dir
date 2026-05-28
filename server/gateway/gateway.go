// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package gateway hosts the in-process grpc-gateway sidecar that exposes
// annotated gRPC services as HTTP/JSON.
//
// The Agent Finder Specification (§3.5, §7) mandates that every Agent
// Registry expose a standard HTTP REST search interface. This sidecar is
// the implementation of that mandate.
//
// Design:
//
//   - Runs in-process beside the gRPC server, on its own listen address.
//   - Dials the gRPC server over loopback (insecure on the loopback only,
//     never over the wire) so existing gRPC interceptors (authn, authz,
//     rate limit, logging, metrics) still fire for HTTP-borne requests.
//   - Disabled by default; enabled via config.HTTPGatewayConfig.
//
// Security notes:
//
//   - The loopback gRPC dial uses insecure credentials. This is safe
//     because the connection is to 127.0.0.1 on the same host; it never
//     traverses the network. Authentication, authorization, and rate
//     limiting are applied by the upstream gRPC interceptors after the
//     gateway re-marshals the HTTP request as a gRPC call.
//
//   - HTTP timeouts mirror the metrics server to defend against slow-loris
//     and connection-exhaustion DoS.
package gateway

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/utils/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed static
var staticFS embed.FS

var logger = logging.Logger("gateway")

const (
	httpReadTimeout       = 10 * time.Second
	httpReadHeaderTimeout = 5 * time.Second
	httpWriteTimeout      = 30 * time.Second
	httpIdleTimeout       = 60 * time.Second
)

// Server is the HTTP-side counterpart to the gRPC Server. It owns the
// gateway mux, the loopback gRPC client conn, and the HTTP listener.
type Server struct {
	httpServer *http.Server
	grpcConn   *grpc.ClientConn
	address    string
}

// Options configures the gateway sidecar.
type Options struct {
	// HTTPAddress is the address the HTTP gateway will bind to (e.g. ":8889").
	// Required.
	HTTPAddress string

	// GRPCEndpoint is the loopback address of the gRPC server the gateway
	// proxies to (e.g. "127.0.0.1:8888"). Required.
	GRPCEndpoint string

	// RegisterHandlers is invoked with the gateway mux and the loopback
	// gRPC client conn. Implementations call e.g.
	//
	//   catalogv1.RegisterAgentFinderServiceHandler(ctx, mux, conn)
	//
	// for each annotated service they want exposed over HTTP. Required.
	RegisterHandlers func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

// New constructs a gateway server. The HTTP listener is created but not
// yet started; call Start to begin serving.
func New(ctx context.Context, opts Options) (*Server, error) {
	if opts.HTTPAddress == "" {
		return nil, fmt.Errorf("gateway http address is required")
	}

	if opts.GRPCEndpoint == "" {
		return nil, fmt.Errorf("gateway grpc endpoint is required")
	}

	if opts.RegisterHandlers == nil {
		return nil, fmt.Errorf("gateway register handler function is required")
	}

	// Loopback dial only — gateway always connects to 127.0.0.1 so
	// insecure credentials are appropriate. Connections to the gRPC
	// server from off-host clients continue to go through whatever
	// transport credentials the gRPC server is configured with.
	//
	// We use grpc.NewClient (non-blocking) so the gateway can be
	// constructed before the gRPC server has started accepting
	// connections — the actual TCP dial happens lazily on the first
	// proxied request.
	conn, err := grpc.NewClient(
		opts.GRPCEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %q: %w", opts.GRPCEndpoint, err)
	}

	mux := runtime.NewServeMux(
		// UseProtoNames keeps the wire JSON in snake_case (what the spec
		// shows in its examples); EmitUnpopulated ensures clients see a
		// consistent shape even when optional fields are missing.
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	if err := opts.RegisterHandlers(ctx, mux, conn); err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("failed to register grpc-gateway handlers: %w", err)
	}

	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("failed to load embedded static files: %w", err)
	}

	handler := withStaticFallback(mux, staticContent)

	httpServer := &http.Server{
		Addr:              opts.HTTPAddress,
		Handler:           handler,
		ReadTimeout:       httpReadTimeout,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		grpcConn:   conn,
		address:    opts.HTTPAddress,
	}, nil
}

// Start binds the configured listen address and serves the HTTP gateway
// in a background goroutine. Binding is synchronous so immediate failures
// (port in use, bind permission denied) surface from Start directly
// instead of disappearing into a log entry the operator may never see.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address) //nolint:noctx
	if err != nil {
		return fmt.Errorf("failed to listen on %q: %w", s.address, err)
	}

	go func() {
		logger.Info("HTTP gateway serving", "address", s.address)

		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP gateway error", "error", err)
		}
	}()

	logger.Info("HTTP gateway started", "address", s.address)

	return nil
}

// Stop gracefully shuts the gateway and closes the loopback connection.
func (s *Server) Stop(ctx context.Context) error {
	logger.Info("Stopping HTTP gateway", "address", s.address)

	var firstErr error

	if err := s.httpServer.Shutdown(ctx); err != nil {
		firstErr = fmt.Errorf("failed to shutdown HTTP gateway: %w", err)
	}

	if err := s.grpcConn.Close(); err != nil && firstErr == nil {
		firstErr = fmt.Errorf("failed to close gateway gRPC conn: %w", err)
	}

	if firstErr == nil {
		logger.Info("HTTP gateway stopped")
	}

	return firstErr
}

// RegisterAgentFinder is a convenience helper that callers can pass as
// Options.RegisterHandlers to expose just the Agent Finder service.
//
// Additional services can be wired through a custom RegisterHandlers
// closure without modifying this package.
func RegisterAgentFinder(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if err := catalogv1.RegisterAgentFinderServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register AgentFinderService handler: %w", err)
	}

	return nil
}

// withStaticFallback wraps the gRPC-gateway mux so that requests not
// matching any registered RPC pattern are served from the embedded
// static filesystem. The root path "/" serves index.html.
func withStaticFallback(mux *runtime.ServeMux, static fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			data, err := fs.ReadFile(static, "index.html")
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)

				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)

			return
		}

		// For any path that exists in the static FS, serve it directly.
		if name := r.URL.Path[1:]; name != "" {
			if _, err := fs.Stat(static, name); err == nil {
				http.ServeFileFS(w, r, static, name)

				return
			}
		}

		// Otherwise delegate to the gRPC-gateway mux (API routes).
		mux.ServeHTTP(w, r)
	})
}
