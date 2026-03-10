// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/agntcy/dir/auth/authzserver"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gopkg.in/yaml.v3"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))
	slog.SetDefault(logger)

	configPath := getEnv("CONFIG_PATH", "/etc/envoy-authz/config.yaml")

	oidcConfig, err := loadOIDCConfig(configPath)
	if err != nil {
		logger.Error("failed to load OIDC config", "path", configPath, "error", err)
		os.Exit(1)
	}

	logger.Info("loaded OIDC config",
		"roles", len(oidcConfig.Roles),
		"publicPaths", len(oidcConfig.PublicPaths),
		"denyListSize", len(oidcConfig.UserDenyList),
	)

	authzServer, err := authzserver.NewOIDCAuthorizationServer(oidcConfig, logger)
	if err != nil {
		logger.Error("failed to create authorization server", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	authv3.RegisterAuthorizationServer(grpcServer, authzServer)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	listenAddr := getEnv("LISTEN_ADDRESS", ":9002")

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", listenAddr)
	if err != nil {
		logger.Error("failed to listen", "address", listenAddr, "error", err)
		os.Exit(1)
	}

	logger.Info("starting OIDC authorization server", "address", listenAddr)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down gracefully...")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

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

func loadOIDCConfig(path string) (*authzserver.OIDCConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config authzserver.OIDCConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}
