// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package v1

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestWorkload_DeepCopy(t *testing.T) {
	t.Run("nil workload returns nil", func(t *testing.T) {
		var w *Workload
		result := w.DeepCopy()
		if result != nil {
			t.Errorf("DeepCopy() of nil = %v, want nil", result)
		}
	})

	t.Run("copies basic fields", func(t *testing.T) {
		original := &Workload{
			Id:       "test-id",
			Name:     "test-name",
			Hostname: "test-host",
			Runtime:  "docker",
			Type:     "container",
		}

		copied := original.DeepCopy()

		if copied.Id != original.Id {
			t.Errorf("Id = %v, want %v", copied.Id, original.Id)
		}
		if copied.Name != original.Name {
			t.Errorf("Name = %v, want %v", copied.Name, original.Name)
		}
		if copied.Hostname != original.Hostname {
			t.Errorf("Hostname = %v, want %v", copied.Hostname, original.Hostname)
		}
		if copied.Runtime != original.Runtime {
			t.Errorf("Runtime = %v, want %v", copied.Runtime, original.Runtime)
		}
		if copied.Type != original.Type {
			t.Errorf("Type = %v, want %v", copied.Type, original.Type)
		}
	})

	t.Run("copies are independent - modifying copy does not affect original", func(t *testing.T) {
		original := &Workload{
			Id:        "test-id",
			Name:      "test-name",
			Labels:    map[string]string{"key": "value"},
			Addresses: []string{"addr1", "addr2"},
		}

		copied := original.DeepCopy()

		// Modify the copy
		copied.Id = "modified-id"
		copied.Labels["key"] = "modified"
		copied.Labels["new"] = "newvalue"
		copied.Addresses[0] = "modified-addr"

		// Original should be unchanged
		if original.Id != "test-id" {
			t.Errorf("Original Id was modified: %v", original.Id)
		}
		if original.Labels["key"] != "value" {
			t.Errorf("Original Labels was modified: %v", original.Labels)
		}
		if _, exists := original.Labels["new"]; exists {
			t.Error("Original Labels has new key that shouldn't exist")
		}
		if original.Addresses[0] != "addr1" {
			t.Errorf("Original Addresses was modified: %v", original.Addresses)
		}
	})

	t.Run("copies slices", func(t *testing.T) {
		original := &Workload{
			Addresses:       []string{"addr1", "addr2"},
			Ports:           []string{"8080", "443"},
			IsolationGroups: []string{"network1"},
		}

		copied := original.DeepCopy()

		if len(copied.Addresses) != len(original.Addresses) {
			t.Errorf("Addresses length = %d, want %d", len(copied.Addresses), len(original.Addresses))
		}
		if len(copied.Ports) != len(original.Ports) {
			t.Errorf("Ports length = %d, want %d", len(copied.Ports), len(original.Ports))
		}
		if len(copied.IsolationGroups) != len(original.IsolationGroups) {
			t.Errorf("IsolationGroups length = %d, want %d", len(copied.IsolationGroups), len(original.IsolationGroups))
		}
	})

	t.Run("copies maps", func(t *testing.T) {
		original := &Workload{
			Labels: map[string]string{
				"app":  "test",
				"env":  "prod",
				"tier": "backend",
			},
			Annotations: map[string]string{
				"note": "important",
			},
		}

		copied := original.DeepCopy()

		if len(copied.Labels) != len(original.Labels) {
			t.Errorf("Labels length = %d, want %d", len(copied.Labels), len(original.Labels))
		}
		for k, v := range original.Labels {
			if copied.Labels[k] != v {
				t.Errorf("Labels[%s] = %v, want %v", k, copied.Labels[k], v)
			}
		}

		if len(copied.Annotations) != len(original.Annotations) {
			t.Errorf("Annotations length = %d, want %d", len(copied.Annotations), len(original.Annotations))
		}
	})

	t.Run("copies services", func(t *testing.T) {
		a2aData, _ := structpb.NewStruct(map[string]any{
			"name":    "agent",
			"version": "1.0",
		})

		original := &Workload{
			Id: "test",
			Services: &WorkloadServices{
				A2A: a2aData,
			},
		}

		copied := original.DeepCopy()

		if copied.Services == nil {
			t.Fatal("Services is nil in copy")
		}
		if copied.Services.A2A == nil {
			t.Fatal("Services.A2A is nil in copy")
		}

		// Verify values
		if copied.Services.A2A.GetFields()["name"].GetStringValue() != "agent" {
			t.Error("Services.A2A.name not copied correctly")
		}
	})
}

func TestRuntimeType_GetName(t *testing.T) {
	t.Run("returns correct name for enum values", func(t *testing.T) {
		tests := []struct {
			input    RuntimeType
			expected string
		}{
			{RuntimeType_RUNTIME_TYPE_UNSPECIFIED, "unspecified"},
			{RuntimeType_RUNTIME_TYPE_DOCKER, "docker"},
			{RuntimeType_RUNTIME_TYPE_KUBERNETES, "kubernetes"},
		}

		for _, test := range tests {
			if test.input.GetName() != test.expected {
				t.Errorf("GetName() = %v, want %v", test.input.GetName(), test.expected)
			}
		}
	})
}

func TestWorkloadType_GetName(t *testing.T) {
	t.Run("returns correct name for enum values", func(t *testing.T) {
		tests := []struct {
			input    WorkloadType
			expected string
		}{
			{WorkloadType_WORKLOAD_TYPE_UNSPECIFIED, "unspecified"},
			{WorkloadType_WORKLOAD_TYPE_PROCESS, "process"},
			{WorkloadType_WORKLOAD_TYPE_CONTAINER, "container"},
			{WorkloadType_WORKLOAD_TYPE_POD, "pod"},
		}

		for _, test := range tests {
			if test.input.GetName() != test.expected {
				t.Errorf("GetName() = %v, want %v", test.input.GetName(), test.expected)
			}
		}
	})
}
