// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:modernize,godot,nlreturn,wsl_v5
package database

import (
	"context"
	"testing"

	v1 "github.com/agntcy/dir/runtime/api/runtime/v1"
)

// mockStore implements storetypes.StoreReader for testing
type mockStore struct {
	workloads []*v1.Workload
	listErr   error
}

func (m *mockStore) ListWorkloadIDs(ctx context.Context) (map[string]struct{}, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	ids := make(map[string]struct{})
	for _, w := range m.workloads {
		ids[w.GetId()] = struct{}{}
	}
	return ids, nil
}

func (m *mockStore) ListWorkloads(ctx context.Context) ([]*v1.Workload, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.workloads, nil
}

func (m *mockStore) WatchWorkloads(ctx context.Context, handler func(workload *v1.Workload, deleted bool)) error {
	return nil
}

func (m *mockStore) Close() error {
	return nil
}

func newTestDatabase(workloads []*v1.Workload) *Database {
	return &Database{
		store: &mockStore{workloads: workloads},
	}
}

func TestDatabase_Count(t *testing.T) {
	tests := []struct {
		name      string
		workloads []*v1.Workload
		want      int
	}{
		{
			name:      "empty store returns 0",
			workloads: []*v1.Workload{},
			want:      0,
		},
		{
			name: "returns correct count",
			workloads: []*v1.Workload{
				{Id: "1"},
				{Id: "2"},
				{Id: "3"},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDatabase(tt.workloads)
			if got := db.Count(context.Background()); got != tt.want {
				t.Errorf("Count() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatabase_Get(t *testing.T) {
	workloads := []*v1.Workload{
		{Id: "id-1", Name: "workload-1", Hostname: "host-1"},
		{Id: "id-2", Name: "workload-2", Hostname: "host-2"},
	}
	db := newTestDatabase(workloads)

	tests := []struct {
		name    string
		id      string
		wantID  string
		wantErr bool
	}{
		{
			name:    "finds by ID",
			id:      "id-1",
			wantID:  "id-1",
			wantErr: false,
		},
		{
			name:    "finds by Name",
			id:      "workload-2",
			wantID:  "id-2",
			wantErr: false,
		},
		{
			name:    "finds by Hostname",
			id:      "host-1",
			wantID:  "id-1",
			wantErr: false,
		},
		{
			name:    "returns error for not found",
			id:      "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.Get(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.GetId() != tt.wantID {
				t.Errorf("Get() ID = %v, want %v", got.GetId(), tt.wantID)
			}
		})
	}
}

func TestDatabase_List(t *testing.T) {
	workloads := []*v1.Workload{
		{Id: "1", Labels: map[string]string{"app": "web", "env": "prod"}},
		{Id: "2", Labels: map[string]string{"app": "api", "env": "prod"}},
		{Id: "3", Labels: map[string]string{"app": "web", "env": "dev"}},
		{Id: "4", Labels: map[string]string{"app": "db"}},
	}
	db := newTestDatabase(workloads)

	tests := []struct {
		name    string
		filters map[string]string
		wantIDs []string
	}{
		{
			name:    "no filter returns all",
			filters: nil,
			wantIDs: []string{"1", "2", "3", "4"},
		},
		{
			name:    "empty filter returns all",
			filters: map[string]string{},
			wantIDs: []string{"1", "2", "3", "4"},
		},
		{
			name:    "filter by single label",
			filters: map[string]string{"app": "web"},
			wantIDs: []string{"1", "3"},
		},
		{
			name:    "filter by multiple labels",
			filters: map[string]string{"app": "web", "env": "prod"},
			wantIDs: []string{"1"},
		},
		{
			name:    "filter with no matches",
			filters: map[string]string{"app": "nonexistent"},
			wantIDs: []string{},
		},
		{
			name:    "filter by label that doesn't exist on all",
			filters: map[string]string{"env": "prod"},
			wantIDs: []string{"1", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.List(context.Background(), tt.filters)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			gotIDs := make([]string, len(got))
			for i, w := range got {
				gotIDs[i] = w.GetId()
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Errorf("List() returned %d items, want %d", len(gotIDs), len(tt.wantIDs))
				return
			}

			// Check all expected IDs are present
			for _, wantID := range tt.wantIDs {
				found := false
				for _, gotID := range gotIDs {
					if gotID == wantID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("List() missing expected ID %s", wantID)
				}
			}
		})
	}
}

func TestMatchesLabelFilter(t *testing.T) {
	tests := []struct {
		name     string
		workload *v1.Workload
		filters  map[string]string
		want     bool
	}{
		{
			name:     "empty filter matches any workload",
			workload: &v1.Workload{Labels: map[string]string{"key": "value"}},
			filters:  map[string]string{},
			want:     true,
		},
		{
			name:     "exact match",
			workload: &v1.Workload{Labels: map[string]string{"key": "value"}},
			filters:  map[string]string{"key": "value"},
			want:     true,
		},
		{
			name:     "no match on value",
			workload: &v1.Workload{Labels: map[string]string{"key": "value"}},
			filters:  map[string]string{"key": "other"},
			want:     false,
		},
		{
			name:     "no match when key missing",
			workload: &v1.Workload{Labels: map[string]string{"key": "value"}},
			filters:  map[string]string{"missing": "value"},
			want:     false,
		},
		{
			name:     "regex pattern match",
			workload: &v1.Workload{Labels: map[string]string{"version": "v1.2.3"}},
			filters:  map[string]string{"version": "v1\\..*"},
			want:     true,
		},
		{
			name:     "regex pattern no match",
			workload: &v1.Workload{Labels: map[string]string{"version": "v2.0.0"}},
			filters:  map[string]string{"version": "v1\\..*"},
			want:     false,
		},
		{
			name:     "workload with nil labels",
			workload: &v1.Workload{Labels: nil},
			filters:  map[string]string{"key": "value"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesLabelFilter(tt.workload, tt.filters); got != tt.want {
				t.Errorf("matchesLabelFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
