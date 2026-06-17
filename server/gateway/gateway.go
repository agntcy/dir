// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package gateway hosts the in-process grpc-gateway sidecar that exposes
// annotated gRPC services as HTTP/JSON. It dials the gRPC server over
// loopback so existing interceptors (authn, authz, rate limit, logging,
// metrics) still apply to HTTP requests. Disabled by default.
package gateway

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/utils/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed all:static
var staticFS embed.FS

var logger = logging.Logger("gateway")

const (
	httpReadTimeout       = 10 * time.Second
	httpReadHeaderTimeout = 5 * time.Second
	httpWriteTimeout      = 30 * time.Second
	httpIdleTimeout       = 60 * time.Second
)

// Server owns the gateway mux, the loopback gRPC client conn, and the HTTP listener.
type Server struct {
	httpServer *http.Server
	grpcConn   *grpc.ClientConn
	address    string
}

// UIConfig holds runtime configuration for the embedded catalog UI.
type UIConfig struct {
	CatalogTitle string
}

// Options configures the gateway sidecar.
type Options struct {
	// HTTPAddress is the address the HTTP gateway binds to (e.g. ":8889").
	HTTPAddress string

	// GRPCEndpoint is the loopback address of the gRPC server to proxy to.
	GRPCEndpoint string

	// GRPCDialOptions are additional options to use when dialing the gRPC server.
	GRPCDialOptions []grpc.DialOption

	// RegisterHandlers registers each annotated service with the gateway mux
	// (e.g. RegisterAIFinder).
	RegisterHandlers func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error

	// UIConfig supplies deployment-specific settings for the embedded catalog UI.
	UIConfig UIConfig
}

// New constructs a gateway server. Call Start to begin serving.
func New(ctx context.Context, opts Options) (*Server, error) {
	if opts.HTTPAddress == "" {
		return nil, errors.New("gateway http address is required")
	}

	if opts.GRPCEndpoint == "" {
		return nil, errors.New("gateway grpc endpoint is required")
	}

	if opts.RegisterHandlers == nil {
		return nil, errors.New("gateway register handlers function is required")
	}

	if len(opts.GRPCDialOptions) == 0 {
		opts.GRPCDialOptions = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
	}

	// Loopback dial only, so insecure credentials are appropriate. grpc.NewClient
	// is non-blocking, so the gateway can be constructed before the gRPC server
	// accepts connections.
	conn, err := grpc.NewClient(opts.GRPCEndpoint, opts.GRPCDialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %q: %w", opts.GRPCEndpoint, err)
	}

	// Field names use protojson's lowerCamelCase default (specVersion, mediaType)
	// to match the AI Catalog spec.
	jsonMarshaler := &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames:   false,
			EmitUnpopulated: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}

	// HTTPBodyMarshaler emits google.api.HttpBody payloads as raw bytes with the
	// supplied Content-Type (for export-style endpoints) and falls back to JSON
	// for all other messages.
	httpBodyMarshaler := &runtime.HTTPBodyMarshaler{Marshaler: jsonMarshaler}

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, httpBodyMarshaler),
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

	handler := withStaticFallback(mux, staticContent, opts.UIConfig)

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

// Start binds the listen address and serves in a background goroutine. Binding
// is synchronous so immediate failures (port in use) surface from Start.
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

// RegisterAIFinder is a RegisterHandlers implementation that exposes the
// AI Finder service over HTTP.
func RegisterAIFinder(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if err := catalogv1.RegisterAIFinderServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register AIFinderService handler: %w", err)
	}

	return nil
}

// withStaticFallback wraps the gRPC-gateway mux so that requests not
// matching any registered RPC pattern are served from the embedded
// static filesystem. The root path "/" serves index.html. Paths starting
// with /v1/ or /.well-known/ are delegated to the gRPC-gateway mux.
// All other paths that don't match a static file serve index.html to
// support SPA client-side routing.
func withStaticFallback(mux *runtime.ServeMux, static fs.FS, ui UIConfig) http.Handler {
	indexHTML, _ := fs.ReadFile(static, "index.html")
	uiConfigPayload := uiConfigJSON(ui)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/ui/config.json" {
			serveUIConfig(w, r, uiConfigPayload)

			return
		}

		// API routes go straight to the gRPC-gateway mux.
		if len(path) >= 4 && path[:4] == "/v1/" ||
			len(path) >= 13 && path[:13] == "/.well-known/" {
			mux.ServeHTTP(w, r)

			return
		}

		// Root or explicit index.html.
		if path == "/" || path == "/index.html" {
			serveIndexHTML(w, r, indexHTML)

			return
		}

		// Serve static file if it exists (JS, CSS, images, etc).
		if name := path[1:]; name != "" && tryServeStatic(w, r, static, path, name) {
			return
		}

		// SvelteKit may request /route/__data.json during client navigations. Do not
		// SPA-fallback with HTML here — that JSON parse failure surfaces as a 500 page.
		if strings.HasSuffix(path, "/__data.json") {
			http.NotFound(w, r)

			return
		}

		// SPA fallback: serve index.html for client-side routing.
		serveIndexHTML(w, r, indexHTML)
	})
}

// tryServeStatic serves a static asset when possible. It returns true when the
// request is fully handled (including 404/500). Benign non-file paths (e.g.
// trailing-slash routes) return false so the caller can SPA-fallback.
func tryServeStatic(w http.ResponseWriter, r *http.Request, static fs.FS, path, name string) bool {
	if isStaticPathTraversal(name) {
		http.NotFound(w, r)

		return true
	}

	if !fs.ValidPath(name) {
		return false
	}

	err := serveStaticFile(w, r, static, name)
	if err == nil {
		return true
	}

	if !errors.Is(err, fs.ErrNotExist) {
		logger.Error("failed to serve static file", "path", path, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return true
	}

	if strings.HasPrefix(path, "/_app/") {
		http.NotFound(w, r)

		return true
	}

	return false
}
