// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package healthcheck provides gRPC health check service.
package healthcheck

import (
	"context"
	"sync"
	"time"

	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var logger = logging.Logger("healthcheck")

// CheckFunc is a function that performs a health check.
// Return true if healthy, false otherwise.
type CheckFunc func(ctx context.Context) bool

// Checker manages health checks using gRPC health checking protocol.
type Checker struct {
	mu              sync.RWMutex
	readinessChecks map[string]CheckFunc
	healthServer    *health.Server
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// New creates a new health checker.
func New() *Checker {
	return &Checker{
		readinessChecks: make(map[string]CheckFunc),
		healthServer:    health.NewServer(),
		stopChan:        make(chan struct{}),
	}
}

// AddReadinessCheck adds a readiness check.
func (c *Checker) AddReadinessCheck(name string, check CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.readinessChecks[name] = check
}

// Register registers the health service with the gRPC server.
func (c *Checker) Register(grpcServer *grpc.Server) {
	grpc_health_v1.RegisterHealthServer(grpcServer, c.healthServer)
	logger.Info("Registered gRPC health service")
}

// Start starts the health check monitoring.
// It periodically checks all registered readiness checks and updates the health status.
func (c *Checker) Start(ctx context.Context) error {
	// Set initial status as NOT_SERVING
	c.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Start background goroutine to monitor health checks
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		c.monitorHealth(ctx)
	}()

	logger.Info("Health check monitoring started")

	return nil
}

// Stop gracefully stops the health check monitoring.
func (c *Checker) Stop(ctx context.Context) error {
	logger.Info("Stopping health check monitoring")

	// Signal stop and wait for goroutine to finish
	close(c.stopChan)
	c.wg.Wait()

	// Set status as not serving
	c.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	return nil
}

// monitorHealth continuously monitors health checks and updates the health status.
func (c *Checker) monitorHealth(ctx context.Context) {
	//nolint:mnd
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.updateHealthStatus(ctx)
		}
	}
}

// updateHealthStatus runs all readiness checks and updates the health status.
func (c *Checker) updateHealthStatus(ctx context.Context) {
	c.mu.RLock()

	checks := make(map[string]CheckFunc, len(c.readinessChecks))
	for name, check := range c.readinessChecks {
		checks[name] = check
	}

	c.mu.RUnlock()

	// Run all checks with timeout
	//nolint:mnd
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	allHealthy := true
	failedChecks := []string{}

	for name, check := range checks {
		if !check(checkCtx) {
			allHealthy = false

			failedChecks = append(failedChecks, name)
		}
	}

	if allHealthy {
		c.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	} else {
		logger.Warn("Health checks failed", "failed_checks", failedChecks)
		c.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}
}
