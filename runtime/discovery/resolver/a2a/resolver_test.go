// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:errcheck
package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
)

func TestNewResolver(t *testing.T) {
	cfg := Config{
		Enabled:    true,
		Timeout:    5 * time.Second,
		LabelKey:   "test-key",
		LabelValue: "test-value",
		Paths:      []string{"/.well-known/agent-card.json"},
	}

	res := NewResolver(cfg)
	if res == nil {
		t.Fatal("NewResolver returned nil")
	}

	r, ok := res.(*resolver)
	if !ok {
		t.Fatal("NewResolver did not return *resolver")
	}

	if r.timeout != cfg.Timeout {
		t.Errorf("timeout = %v, want %v", r.timeout, cfg.Timeout)
	}

	if r.labelKey != cfg.LabelKey {
		t.Errorf("labelKey = %v, want %v", r.labelKey, cfg.LabelKey)
	}

	if r.labelValue != cfg.LabelValue {
		t.Errorf("labelValue = %v, want %v", r.labelValue, cfg.LabelValue)
	}
}

func TestResolver_Name(t *testing.T) {
	r := NewResolver(Config{})
	if r.Name() != ResolverType {
		t.Errorf("Name() = %v, want %v", r.Name(), ResolverType)
	}
}

func TestResolver_CanResolve(t *testing.T) {
	r := NewResolver(Config{
		LabelKey:   "org.agntcy/agent-type",
		LabelValue: "a2a",
	})

	tests := []struct {
		name     string
		workload *runtimev1.Workload
		want     bool
	}{
		{
			name: "can resolve workload with matching label and addresses/ports",
			workload: &runtimev1.Workload{
				Labels:    map[string]string{"org.agntcy/agent-type": "a2a"},
				Addresses: []string{"localhost"},
				Ports:     []string{"8080"},
			},
			want: true,
		},
		{
			name: "cannot resolve workload without label",
			workload: &runtimev1.Workload{
				Labels:    map[string]string{},
				Addresses: []string{"localhost"},
				Ports:     []string{"8080"},
			},
			want: false,
		},
		{
			name: "cannot resolve workload with wrong label value",
			workload: &runtimev1.Workload{
				Labels:    map[string]string{"org.agntcy/agent-type": "other"},
				Addresses: []string{"localhost"},
				Ports:     []string{"8080"},
			},
			want: false,
		},
		{
			name: "cannot resolve workload without addresses",
			workload: &runtimev1.Workload{
				Labels:    map[string]string{"org.agntcy/agent-type": "a2a"},
				Addresses: []string{},
				Ports:     []string{"8080"},
			},
			want: false,
		},
		{
			name: "cannot resolve workload without ports",
			workload: &runtimev1.Workload{
				Labels:    map[string]string{"org.agntcy/agent-type": "a2a"},
				Addresses: []string{"localhost"},
				Ports:     []string{},
			},
			want: false,
		},
		{
			name: "case insensitive label value match",
			workload: &runtimev1.Workload{
				Labels:    map[string]string{"org.agntcy/agent-type": "A2A"},
				Addresses: []string{"localhost"},
				Ports:     []string{"8080"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.CanResolve(tt.workload); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_Resolve(t *testing.T) {
	// Create a test server that returns a valid agent card
	agentCard := map[string]any{
		"name":        "test-agent",
		"version":     "1.0.0",
		"description": "A test agent",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent-card.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(agentCard)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Extract host and port from test server
	// server.URL is like "http://127.0.0.1:12345"
	r := NewResolver(Config{
		Timeout:    5 * time.Second,
		LabelKey:   "test",
		LabelValue: "true",
		Paths:      []string{"/.well-known/agent-card.json"},
	})

	t.Run("resolves successfully from valid endpoint", func(t *testing.T) {
		// We need to use the test server's actual address
		// Parse the URL to get host:port
		workload := &runtimev1.Workload{
			Id:        "test-workload",
			Labels:    map[string]string{"test": "true"},
			Addresses: []string{"127.0.0.1"},
			Ports:     []string{server.URL[len("http://127.0.0.1:"):]}, // Extract port
		}

		ctx := context.Background()

		result, err := r.Resolve(ctx, workload)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result is not map[string]any: %T", result)
		}

		if resultMap["name"] != "test-agent" {
			t.Errorf("Result name = %v, want 'test-agent'", resultMap["name"])
		}
	})

	t.Run("returns error when no endpoints reachable", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:        "test-workload",
			Labels:    map[string]string{"test": "true"},
			Addresses: []string{"127.0.0.1"},
			Ports:     []string{"99999"}, // Invalid port
		}

		ctx := context.Background()

		_, err := r.Resolve(ctx, workload)
		if err == nil {
			t.Error("Resolve() should return error for unreachable endpoint")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:        "test-workload",
			Labels:    map[string]string{"test": "true"},
			Addresses: []string{"127.0.0.1"},
			Ports:     []string{"8080"},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := r.Resolve(ctx, workload)
		if err == nil {
			t.Error("Resolve() should return error for cancelled context")
		}
	})
}

func TestResolver_Apply(t *testing.T) {
	r := NewResolver(Config{})

	t.Run("applies result to workload", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id: "test-workload",
		}

		result := map[string]any{
			"name":    "test-agent",
			"version": "1.0.0",
		}

		ctx := context.Background()

		err := r.Apply(ctx, workload, result)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}

		if workload.GetServices() == nil {
			t.Fatal("Services is nil after Apply")
		}

		if workload.GetServices().GetA2A() == nil {
			t.Fatal("Services.A2A is nil after Apply")
		}

		fields := workload.GetServices().GetA2A().GetFields()
		if fields["name"].GetStringValue() != "test-agent" {
			t.Errorf("A2A name = %v, want 'test-agent'", fields["name"].GetStringValue())
		}
	})

	t.Run("preserves existing services", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:       "test-workload",
			Services: &runtimev1.WorkloadServices{},
		}

		result := map[string]any{
			"name": "test-agent",
		}

		ctx := context.Background()

		err := r.Apply(ctx, workload, result)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}

		if workload.GetServices().GetA2A() == nil {
			t.Fatal("Services.A2A is nil after Apply")
		}
	})
}
