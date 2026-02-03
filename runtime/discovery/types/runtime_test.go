// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"testing"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
)

func TestRuntimeEventType_Constants(t *testing.T) {
	// Verify event type constants are defined correctly
	tests := []struct {
		eventType RuntimeEventType
		want      string
	}{
		{RuntimeEventTypeAdded, "added"},
		{RuntimeEventTypeModified, "modified"},
		{RuntimeEventTypeDeleted, "deleted"},
		{RuntimeEventTypePaused, "paused"},
	}

	for _, tt := range tests {
		if string(tt.eventType) != tt.want {
			t.Errorf("RuntimeEventType = %v, want %v", tt.eventType, tt.want)
		}
	}
}

func TestRuntimeEvent_Fields(t *testing.T) {
	workload := &runtimev1.Workload{
		Id:   "test-id",
		Name: "test-name",
	}

	event := &RuntimeEvent{
		Type:     RuntimeEventTypeAdded,
		Workload: workload,
	}

	if event.Type != RuntimeEventTypeAdded {
		t.Errorf("Event.Type = %v, want %v", event.Type, RuntimeEventTypeAdded)
	}

	if event.Workload == nil {
		t.Error("Event.Workload is nil")
	}

	if event.Workload.GetId() != "test-id" {
		t.Errorf("Event.Workload.Id = %v, want 'test-id'", event.Workload.GetId())
	}
}
