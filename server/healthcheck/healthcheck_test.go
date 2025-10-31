// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:noctx,goconst
package healthcheck

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestNew(t *testing.T) {
	checker := New()
	if checker == nil {
		t.Fatal("Expected non-nil checker")
	}

	if checker.readinessChecks == nil {
		t.Fatal("Expected readinessChecks map to be initialized")
	}

	if checker.healthServer == nil {
		t.Fatal("Expected healthServer to be initialized")
	}
}

func TestAddReadinessCheck(t *testing.T) {
	checker := New()

	checkCalled := false
	testCheck := func(ctx context.Context) bool {
		checkCalled = true

		return true
	}

	checker.AddReadinessCheck("test", testCheck)

	// Verify check was added
	checker.mu.RLock()

	if _, exists := checker.readinessChecks["test"]; !exists {
		t.Error("Expected readiness check to be added")
	}

	checker.mu.RUnlock()

	// Call the check function
	ctx := context.Background()

	result := checker.readinessChecks["test"](ctx)
	if !result {
		t.Error("Expected check to return true")
	}

	if !checkCalled {
		t.Error("Expected check function to be called")
	}
}

func TestStartAndStop(t *testing.T) {
	checker := New()

	// Create a gRPC server and register the health service
	grpcServer := grpc.NewServer()
	checker.Register(grpcServer)

	// Start the gRPC server in the background
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	defer grpcServer.Stop()

	ctx := context.Background()

	err = checker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start health check monitoring: %v", err)
	}

	// Give monitoring time to start
	time.Sleep(100 * time.Millisecond)

	// Stop monitoring
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = checker.Stop(stopCtx)
	if err != nil {
		t.Errorf("Failed to stop health check monitoring: %v", err)
	}
}

func TestStopWithoutStart(t *testing.T) {
	checker := New()

	// Stop without starting should not error
	ctx := context.Background()

	err := checker.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error when stopping without start, got: %v", err)
	}
}

func TestHealthCheckServing(t *testing.T) {
	checker := New()

	// Create a gRPC server and register the health service
	grpcServer := grpc.NewServer()
	checker.Register(grpcServer)

	// Start the gRPC server in the background
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	defer grpcServer.Stop()

	// Connect to the server
	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)

	ctx := context.Background()

	// Start health check monitoring
	err = checker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start health check monitoring: %v", err)
	}

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(stopCtx)
	}()

	// Give monitoring time to start
	time.Sleep(100 * time.Millisecond)

	// Check health status - should be SERVING
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("Failed to check health: %v", err)
	}

	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected status SERVING, got %v", resp.GetStatus())
	}
}

func TestHealthCheckWithFailingCheck(t *testing.T) {
	checker := New()

	// Add a failing check
	checker.AddReadinessCheck("failing", func(ctx context.Context) bool {
		return false
	})

	// Create a gRPC server and register the health service
	grpcServer := grpc.NewServer()
	checker.Register(grpcServer)

	// Start the gRPC server in the background
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	defer grpcServer.Stop()

	// Connect to the server
	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)

	ctx := context.Background()

	// Start health check monitoring
	err = checker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start health check monitoring: %v", err)
	}

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(stopCtx)
	}()

	// Wait for health check to run (checks run every 5 seconds)
	time.Sleep(6 * time.Second)

	// Check health status - should be NOT_SERVING
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("Failed to check health: %v", err)
	}

	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Errorf("Expected status NOT_SERVING, got %v", resp.GetStatus())
	}
}

func TestHealthCheckWithPassingChecks(t *testing.T) {
	checker := New()

	// Add passing checks
	checker.AddReadinessCheck("check1", func(ctx context.Context) bool {
		return true
	})

	checker.AddReadinessCheck("check2", func(ctx context.Context) bool {
		return true
	})

	// Create a gRPC server and register the health service
	grpcServer := grpc.NewServer()
	checker.Register(grpcServer)

	// Start the gRPC server in the background
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	defer grpcServer.Stop()

	// Connect to the server
	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)

	ctx := context.Background()

	// Start health check monitoring
	err = checker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start health check monitoring: %v", err)
	}

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(stopCtx)
	}()

	// Wait for health check to run
	time.Sleep(6 * time.Second)

	// Check health status - should be SERVING
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("Failed to check health: %v", err)
	}

	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected status SERVING, got %v", resp.GetStatus())
	}
}

func TestHealthWatch(t *testing.T) {
	checker := New()

	// Create a gRPC server and register the health service
	grpcServer := grpc.NewServer()
	checker.Register(grpcServer)

	// Start the gRPC server in the background
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	defer grpcServer.Stop()

	// Connect to the server
	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start health check monitoring
	err = checker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start health check monitoring: %v", err)
	}

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(stopCtx)
	}()

	// Watch health status
	stream, err := client.Watch(ctx, &grpc_health_v1.HealthCheckRequest{Service: ""})
	if err != nil {
		t.Fatalf("Failed to watch health: %v", err)
	}

	// Should receive at least one status update
	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive health status: %v", err)
	}

	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Errorf("Expected status SERVING, got %v", resp.GetStatus())
	}
}
