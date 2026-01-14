// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/agntcy/dir/pkg/authprovider"
	"github.com/agntcy/dir/pkg/authprovider/github"
	"github.com/agntcy/dir/pkg/authzserver"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	// Default cache TTL for authentication tokens.
	defaultCacheTTL = 5 * time.Minute

	// Default API timeout for external API calls.
	defaultAPITimeout = 10 * time.Second
)

func main() {
	// Setup logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))
	slog.SetDefault(logger)

	// Load configuration
	config := loadConfig()

	// Initialize providers
	providers := initializeProviders(config)

	if len(providers) == 0 {
		logger.Error("no authentication providers configured")
		os.Exit(1)
	}

	logger.Info("initialized authentication providers",
		"providers", getProviderNames(providers),
		"default", config.DefaultProvider,
	)

	// Create authorization server
	authzConfig := &authzserver.Config{
		DefaultProvider:      config.DefaultProvider,
		AllowedOrgConstructs: parseList(config.AllowedOrgConstructs),
		UserAllowList:        parseList(config.UserAllowList),
		UserDenyList:         parseList(config.UserDenyList),
	}

	authzServer := authzserver.NewAuthorizationServer(providers, authzConfig, logger)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register ext_authz service
	authv3.RegisterAuthorizationServer(grpcServer, authzServer)

	// Register health service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Start server
	listenAddr := config.ListenAddress

	// Use context for listener (noctx)
	listenConfig := &net.ListenConfig{}

	listener, err := listenConfig.Listen(context.Background(), "tcp", listenAddr)
	if err != nil {
		logger.Error("failed to listen", "address", listenAddr, "error", err)
		os.Exit(1)
	}

	logger.Info("starting GitHub authorization server", "address", listenAddr)

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		logger.Info("shutting down gracefully...")
		grpcServer.GracefulStop()
	}()

	// Serve
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

// Config holds service configuration.
type Config struct {
	ListenAddress        string
	DefaultProvider      string
	AllowedOrgConstructs string
	UserAllowList        string
	UserDenyList         string
	CacheTTL             time.Duration

	// Provider-specific configs
	GitHub struct {
		Enabled    bool
		CacheTTL   time.Duration
		APITimeout time.Duration
	}

	// Future: Google, Azure, etc.
}

// loadConfig loads configuration from environment variables.
func loadConfig() *Config {
	config := &Config{
		ListenAddress:        getEnv("LISTEN_ADDRESS", ":9002"),
		DefaultProvider:      getEnv("DEFAULT_PROVIDER", "github"),
		AllowedOrgConstructs: getEnv("ALLOWED_ORG_CONSTRUCTS", ""),
		UserAllowList:        getEnv("USER_ALLOW_LIST", ""),
		UserDenyList:         getEnv("USER_DENY_LIST", ""),
		CacheTTL:             parseDuration(getEnv("CACHE_TTL", "5m"), defaultCacheTTL),
	}

	// GitHub provider config
	config.GitHub.Enabled = getEnv("GITHUB_ENABLED", "true") == "true"
	config.GitHub.CacheTTL = parseDuration(getEnv("GITHUB_CACHE_TTL", "5m"), defaultCacheTTL)
	config.GitHub.APITimeout = parseDuration(getEnv("GITHUB_API_TIMEOUT", "10s"), defaultAPITimeout)

	return config
}

// initializeProviders creates and registers authentication providers.
func initializeProviders(config *Config) map[string]authprovider.Provider {
	providers := make(map[string]authprovider.Provider)

	// GitHub provider
	if config.GitHub.Enabled {
		githubProvider := github.NewProvider(&github.Config{
			CacheTTL:   config.GitHub.CacheTTL,
			APITimeout: config.GitHub.APITimeout,
		})
		providers["github"] = githubProvider

		slog.Info("registered provider", "name", "github")
	}

	// Future providers
	// if config.Google.Enabled {
	//     providers["google"] = google.NewProvider(&google.Config{...})
	// }

	return providers
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

func getLogLevel() slog.Level {
	switch strings.ToLower(getEnv("LOG_LEVEL", "info")) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}

	return defaultValue
}

func parseList(s string) []string {
	if s == "" {
		return []string{}
	}

	parts := strings.Split(s, ",")

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func getProviderNames(providers map[string]authprovider.Provider) []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}

	return names
}
