// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"strings"
	"testing"
)

func TestCrName(t *testing.T) {
	tests := []struct {
		name       string
		workloadID string
		want       string
	}{
		{
			name:       "simple ID",
			workloadID: "abc123",
			want:       "dw-abc123",
		},
		{
			name:       "ID with uppercase",
			workloadID: "ABC123",
			want:       "dw-abc123",
		},
		{
			name:       "ID with underscores",
			workloadID: "my_workload_id",
			want:       "dw-my-workload-id",
		},
		{
			name:       "ID with colons",
			workloadID: "sha256:abc123",
			want:       "dw-sha256-abc123",
		},
		{
			name:       "ID with mixed special chars",
			workloadID: "my_workload:v1.0",
			want:       "dw-my-workload-v1.0",
		},
		{
			name:       "empty ID",
			workloadID: "",
			want:       "dw", // trailing dash is trimmed
		},
		{
			name:       "ID ending with dash after conversion",
			workloadID: "test_",
			want:       "dw-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := crName(tt.workloadID)
			if got != tt.want {
				t.Errorf("crName(%q) = %q, want %q", tt.workloadID, got, tt.want)
			}
		})
	}
}

func TestCrName_MaxLength(t *testing.T) {
	// Kubernetes resource names must be <= 63 characters
	longID := strings.Repeat("a", 100)
	result := crName(longID)

	if len(result) > 63 {
		t.Errorf("crName() returned name longer than 63 chars: len=%d", len(result))
	}
}

func TestCrName_ValidKubernetesName(t *testing.T) {
	testIDs := []string{
		"simple",
		"WITH_UPPER",
		"with:colons",
		"with_underscores",
		"with-dashes",
		"mixed_Case:Special",
		strings.Repeat("x", 100),
	}

	for _, id := range testIDs {
		name := crName(id)

		// Must not exceed 63 chars
		if len(name) > 63 {
			t.Errorf("crName(%q) too long: %d chars", id, len(name))
		}

		// Must be lowercase
		if name != strings.ToLower(name) {
			t.Errorf("crName(%q) is not lowercase: %s", id, name)
		}

		// Must not end with dash
		if strings.HasSuffix(name, "-") {
			t.Errorf("crName(%q) ends with dash: %s", id, name)
		}

		// Must start with dw-
		if !strings.HasPrefix(name, "dw-") {
			t.Errorf("crName(%q) doesn't start with 'dw-': %s", id, name)
		}
	}
}
