// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package main implements a mock Directory server for local testing of Envoy ext_authz.
//
// SECURITY NOTE: This is a TEST/DEVELOPMENT server only.
// It intentionally logs all HTTP headers and user input for debugging purposes.
// DO NOT use this server in production environments.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// Default listen address for the mock server.
	defaultListenAddr = ":8888"

	// HTTP server timeouts.
	serverReadHeaderTimeout = 10 * time.Second
	serverReadTimeout       = 30 * time.Second
	serverWriteTimeout      = 30 * time.Second
	serverIdleTimeout       = 60 * time.Second

	// Graceful shutdown timeout.
	shutdownTimeout = 5 * time.Second
)

func main() {
	addr := os.Getenv("LISTEN_ADDRESS")
	if addr == "" {
		addr = defaultListenAddr
	}

	http.HandleFunc("/", handleRequest)
	http.HandleFunc("/healthz", handleHealth)

	// Create server with timeouts (gosec G114)
	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	//nolint:gosec // G706: Mock server - logs listening address for debugging
	log.Printf("Mock Directory server listening on %s", addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	//nolint:gosec // G110: Mock server for testing - verbose logging intentional
	log.Printf("📨 Request: %s %s", r.Method, r.URL.Path)

	// Log all headers (shows what Envoy adds)
	log.Println("📋 Headers:")

	for name, values := range r.Header {
		for _, value := range values {
			//nolint:gosec // G110,G401: Mock server - logs headers for debugging ext_authz
			log.Printf("  %s: %s", name, value)
		}
	}

	// Extract user info from headers (added by ext_authz)
	authProvider := r.Header.Get("X-Auth-Provider")
	username := r.Header.Get("X-Username")
	userID := r.Header.Get("X-User-Id")
	email := r.Header.Get("X-User-Email")
	orgConstructs := r.Header.Get("X-Org-Constructs")

	// Echo back the request info
	response := map[string]any{
		"message": "Mock Directory API",
		"path":    r.URL.Path,
		"method":  r.Method,
		"authenticated": map[string]string{
			"provider":       authProvider,
			"username":       username,
			"user_id":        userID,
			"email":          email,
			"org_constructs": orgConstructs,
		},
		"note": "This is a mock server for testing ext_authz integration",
	}

	// Pretty print for logs
	if authProvider != "" {
		//nolint:gosec // G110: Mock server - logs authenticated user info for debugging
		log.Printf("✅ Authenticated user: %s (provider: %s, orgs: %s)",
			username, authProvider, orgConstructs)
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"healthy","service":"mock-directory"}`)
}
