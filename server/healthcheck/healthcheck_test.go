// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:noctx,goconst
package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	checker := New()
	if checker == nil {
		t.Fatal("Expected non-nil checker")
	}

	if checker.readinessChecks == nil {
		t.Fatal("Expected readinessChecks map to be initialized")
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

	// Use a high port number unlikely to conflict
	address := "127.0.0.1:58081"

	err := checker.Start(address)
	if err != nil {
		t.Fatalf("Failed to start health check server: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Get actual address after binding
	if checker.server == nil {
		t.Fatal("Expected server to be initialized")
	}

	actualAddr := checker.server.Addr
	if actualAddr == "" {
		t.Fatal("Expected server address to be set")
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = checker.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop health check server: %v", err)
	}
}

func TestStopWithNilServer(t *testing.T) {
	checker := New()

	// Stop without starting should not error
	ctx := context.Background()

	err := checker.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error when stopping with nil server, got: %v", err)
	}
}

func TestHandleLiveness(t *testing.T) {
	checker := New()

	// Use a high port number unlikely to conflict
	address := "127.0.0.1:58082"

	err := checker.Start(address)
	if err != nil {
		t.Fatalf("Failed to start health check server: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Get server address
	baseURL := "http://" + address

	// Test /healthz/live endpoint
	t.Run("healthz_live", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/healthz/live")
		if err != nil {
			t.Fatalf("Failed to request liveness endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var response Response

		err = json.Unmarshal(body, &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Status != "ok" {
			t.Errorf("Expected status 'ok', got %s", response.Status)
		}

		if response.Message != "server is alive" {
			t.Errorf("Expected message 'server is alive', got %s", response.Message)
		}
	})

	// Test /livez endpoint
	t.Run("livez", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/livez")
		if err != nil {
			t.Fatalf("Failed to request liveness endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestHandleReadiness(t *testing.T) {
	checker := New()

	// Add a passing check
	checker.AddReadinessCheck("passing", func(ctx context.Context) bool {
		return true
	})

	// Add a failing check
	checker.AddReadinessCheck("failing", func(ctx context.Context) bool {
		return false
	})

	// Use a high port number unlikely to conflict
	address := "127.0.0.1:58083"

	err := checker.Start(address)
	if err != nil {
		t.Fatalf("Failed to start health check server: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	baseURL := "http://" + address

	// Test /healthz/ready endpoint with failing check
	t.Run("healthz_ready_not_ready", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/healthz/ready")
		if err != nil {
			t.Fatalf("Failed to request readiness endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var response Response

		err = json.Unmarshal(body, &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Status != "not ready" {
			t.Errorf("Expected status 'not ready', got %s", response.Status)
		}

		if response.Checks["passing"] != "pass" {
			t.Errorf("Expected passing check to be 'pass', got %s", response.Checks["passing"])
		}

		if response.Checks["failing"] != "fail" {
			t.Errorf("Expected failing check to be 'fail', got %s", response.Checks["failing"])
		}
	})

	// Test /readyz endpoint
	t.Run("readyz_not_ready", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/readyz")
		if err != nil {
			t.Fatalf("Failed to request readiness endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", resp.StatusCode)
		}
	})
}

func TestHandleReadinessAllPassing(t *testing.T) {
	checker := New()

	// Add only passing checks
	checker.AddReadinessCheck("check1", func(ctx context.Context) bool {
		return true
	})

	checker.AddReadinessCheck("check2", func(ctx context.Context) bool {
		return true
	})

	// Use a high port number unlikely to conflict
	address := "127.0.0.1:58084"

	err := checker.Start(address)
	if err != nil {
		t.Fatalf("Failed to start health check server: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	baseURL := "http://" + address

	resp, err := http.Get(baseURL + "/healthz/ready")
	if err != nil {
		t.Fatalf("Failed to request readiness endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var response Response

	err = json.Unmarshal(body, &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "ready" {
		t.Errorf("Expected status 'ready', got %s", response.Status)
	}

	if response.Checks["check1"] != "pass" {
		t.Errorf("Expected check1 to be 'pass', got %s", response.Checks["check1"])
	}

	if response.Checks["check2"] != "pass" {
		t.Errorf("Expected check2 to be 'pass', got %s", response.Checks["check2"])
	}
}

func TestHandleReadinessNoChecks(t *testing.T) {
	checker := New()

	// Use a high port number unlikely to conflict
	address := "127.0.0.1:58085"

	err := checker.Start(address)
	if err != nil {
		t.Fatalf("Failed to start health check server: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	baseURL := "http://" + address

	resp, err := http.Get(baseURL + "/healthz/ready")
	if err != nil {
		t.Fatalf("Failed to request readiness endpoint: %v", err)
	}
	defer resp.Body.Close()

	// With no checks, should be ready
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var response Response

	err = json.Unmarshal(body, &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "ready" {
		t.Errorf("Expected status 'ready', got %s", response.Status)
	}
}

func TestConcurrentReadinessChecks(t *testing.T) {
	checker := New()

	// Add multiple checks
	for i := range 5 {
		name := fmt.Sprintf("check%d", i)
		checker.AddReadinessCheck(name, func(ctx context.Context) bool {
			return true
		})
	}

	// Use a high port number unlikely to conflict
	address := "127.0.0.1:58086"

	err := checker.Start(address)
	if err != nil {
		t.Fatalf("Failed to start health check server: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = checker.Stop(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	baseURL := "http://" + address

	// Make multiple concurrent requests
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			resp, err := http.Get(baseURL + "/healthz/ready")
			if err != nil {
				t.Errorf("Failed to request readiness endpoint: %v", err)

				done <- false

				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)

				done <- false

				return
			}

			done <- true
		}()
	}

	// Wait for all requests to complete
	for range 10 {
		<-done
	}
}
