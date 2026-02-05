// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oasf

import (
	"context"
	"testing"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
)

func TestResolverType(t *testing.T) {
	if ResolverType != "oasf" {
		t.Errorf("ResolverType = %v, want 'oasf'", ResolverType)
	}
}

// mockResolver for testing Apply without needing a real client.
type mockResolver struct {
	labelKey string
}

func (m *mockResolver) Name() string { return "oasf" }

func (m *mockResolver) CanResolve(workload *runtimev1.Workload) bool {
	if _, hasLabel := workload.GetLabels()[m.labelKey]; !hasLabel {
		return false
	}

	return true
}

func TestResolver_CanResolve(t *testing.T) {
	m := &mockResolver{labelKey: "org.agntcy/agent-record"}

	tests := []struct {
		name     string
		workload *runtimev1.Workload
		want     bool
	}{
		{
			name: "can resolve with label",
			workload: &runtimev1.Workload{
				Labels: map[string]string{"org.agntcy/agent-record": "my-agent:v1"},
			},
			want: true,
		},
		{
			name: "cannot resolve without label",
			workload: &runtimev1.Workload{
				Labels: map[string]string{"other": "label"},
			},
			want: false,
		},
		{
			name: "cannot resolve with nil labels",
			workload: &runtimev1.Workload{
				Labels: nil,
			},
			want: false,
		},
		{
			name: "cannot resolve with empty labels",
			workload: &runtimev1.Workload{
				Labels: map[string]string{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.CanResolve(tt.workload); got != tt.want {
				t.Errorf("CanResolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_Apply(t *testing.T) {
	// Create a minimal resolver just for testing Apply
	r := &resolver{
		labelKey: "org.agntcy/agent-record",
	}

	t.Run("applies result to workload", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id: "test-workload",
		}

		result := map[string]any{
			"cid":  "bafytest123",
			"name": "my-agent:v1",
			"record": map[string]any{
				"name":    "My Agent",
				"version": "1.0.0",
			},
		}

		ctx := context.Background()

		err := r.Apply(ctx, workload, result)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}

		if workload.GetServices() == nil {
			t.Fatal("Services is nil after Apply")
		}

		if workload.GetServices().GetOasf() == nil {
			t.Fatal("Services.Oasf is nil after Apply")
		}

		fields := workload.GetServices().GetOasf().GetFields()
		if fields["cid"].GetStringValue() != "bafytest123" {
			t.Errorf("Oasf cid = %v, want 'bafytest123'", fields["cid"].GetStringValue())
		}
	})

	t.Run("creates services if nil", func(t *testing.T) {
		workload := &runtimev1.Workload{
			Id:       "test-workload",
			Services: nil,
		}

		result := map[string]any{
			"name": "agent",
		}

		ctx := context.Background()

		err := r.Apply(ctx, workload, result)
		if err != nil {
			t.Fatalf("Apply() error = %v", err)
		}

		if workload.GetServices() == nil {
			t.Error("Services should be created")
		}
	})
}
