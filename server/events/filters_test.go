// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"testing"
	"time"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

func TestEventTypeFilter(t *testing.T) {
	filter := EventTypeFilter(
		eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
		eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
	)

	tests := []struct {
		name      string
		eventType eventsv1.EventType
		want      bool
	}{
		{
			name:      "matches first type",
			eventType: eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
			want:      true,
		},
		{
			name:      "matches second type",
			eventType: eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
			want:      true,
		},
		{
			name:      "does not match other type",
			eventType: eventsv1.EventType_EVENT_TYPE_RECORD_DELETED,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				ID:         TestEventID,
				Type:       tt.eventType,
				Timestamp:  time.Now(),
				ResourceID: TestCID123,
			}

			if got := filter(event); got != tt.want {
				t.Errorf("EventTypeFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCIDFilter(t *testing.T) {
	filter := CIDFilter("bafytest123", "bafytest456")

	tests := []struct {
		name       string
		resourceID string
		want       bool
	}{
		{
			name:       "matches first CID",
			resourceID: "bafytest123",
			want:       true,
		},
		{
			name:       "matches second CID",
			resourceID: "bafytest456",
			want:       true,
		},
		{
			name:       "does not match other CID",
			resourceID: "bafytest789",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				ID:         TestEventID,
				Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Timestamp:  time.Now(),
				ResourceID: tt.resourceID,
			}

			if got := filter(event); got != tt.want {
				t.Errorf("CIDFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLabelFilter(t *testing.T) {
	filter := LabelFilter("/skills/AI", "/domains/research")

	tests := []struct {
		name   string
		labels []string
		want   bool
	}{
		{
			name:   "matches first label substring",
			labels: []string{"/skills/AI/ML"},
			want:   true,
		},
		{
			name:   "matches second label substring",
			labels: []string{"/domains/research/quantum"},
			want:   true,
		},
		{
			name:   "matches exact label",
			labels: []string{"/skills/AI"},
			want:   true,
		},
		{
			name:   "matches one of multiple labels",
			labels: []string{"/modules/tensorflow", "/skills/AI/NLP"},
			want:   true,
		},
		{
			name:   "does not match other labels",
			labels: []string{"/modules/pytorch", "/domains/medical"},
			want:   false,
		},
		{
			name:   "empty labels does not match",
			labels: []string{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				ID:         TestEventID,
				Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Timestamp:  time.Now(),
				ResourceID: TestCID123,
				Labels:     tt.labels,
			}

			if got := filter(event); got != tt.want {
				t.Errorf("LabelFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:dupl // Similar test structure to TestAndFilter is intentional
func TestOrFilter(t *testing.T) {
	filter := Or(
		EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED),
		LabelFilter("/skills/AI"),
	)

	tests := []struct {
		name  string
		event *Event
		want  bool
	}{
		{
			name: "matches first filter only",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Labels: []string{"/domains/medical"},
			},
			want: true,
		},
		{
			name: "matches second filter only",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
				Labels: []string{"/skills/AI"},
			},
			want: true,
		},
		{
			name: "matches both filters",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Labels: []string{"/skills/AI"},
			},
			want: true,
		},
		{
			name: "matches neither filter",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_DELETED,
				Labels: []string{"/domains/medical"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.event.ID = TestEventID
			tt.event.Timestamp = time.Now()
			tt.event.ResourceID = TestCID123

			if got := filter(tt.event); got != tt.want {
				t.Errorf("Or() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:dupl // Similar test structure to TestOrFilter is intentional
func TestAndFilter(t *testing.T) {
	filter := And(
		EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED),
		LabelFilter("/skills/AI"),
	)

	tests := []struct {
		name  string
		event *Event
		want  bool
	}{
		{
			name: "matches both filters",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Labels: []string{"/skills/AI"},
			},
			want: true,
		},
		{
			name: "matches first filter only",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Labels: []string{"/domains/medical"},
			},
			want: false,
		},
		{
			name: "matches second filter only",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
				Labels: []string{"/skills/AI"},
			},
			want: false,
		},
		{
			name: "matches neither filter",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_DELETED,
				Labels: []string{"/domains/medical"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.event.ID = TestEventID
			tt.event.Timestamp = time.Now()
			tt.event.ResourceID = TestCID123

			if got := filter(tt.event); got != tt.want {
				t.Errorf("And() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFilter(t *testing.T) {
	filter := Not(EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_DELETED))

	tests := []struct {
		name      string
		eventType eventsv1.EventType
		want      bool
	}{
		{
			name:      "negates matching type",
			eventType: eventsv1.EventType_EVENT_TYPE_RECORD_DELETED,
			want:      false,
		},
		{
			name:      "allows non-matching type",
			eventType: eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				ID:         TestEventID,
				Type:       tt.eventType,
				Timestamp:  time.Now(),
				ResourceID: TestCID123,
			}

			if got := filter(event); got != tt.want {
				t.Errorf("Not() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildFilters(t *testing.T) {
	tests := []struct {
		name    string
		req     *eventsv1.ListenRequest
		wantLen int
	}{
		{
			name: "all filters specified",
			req: &eventsv1.ListenRequest{
				EventTypes:   []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
				CidFilters:   []string{"bafytest123"},
				LabelFilters: []string{"/skills/AI"},
			},
			wantLen: 3,
		},
		{
			name: "only event type filter",
			req: &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
			},
			wantLen: 1,
		},
		{
			name: "only CID filter",
			req: &eventsv1.ListenRequest{
				CidFilters: []string{"bafytest123"},
			},
			wantLen: 1,
		},
		{
			name: "only label filter",
			req: &eventsv1.ListenRequest{
				LabelFilters: []string{"/skills/AI"},
			},
			wantLen: 1,
		},
		{
			name:    "no filters",
			req:     &eventsv1.ListenRequest{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := BuildFilters(tt.req)

			if len(filters) != tt.wantLen {
				t.Errorf("BuildFilters() returned %d filters, want %d", len(filters), tt.wantLen)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	event := &Event{
		ID:         TestEventID,
		Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
		Timestamp:  time.Now(),
		ResourceID: "bafytest123",
		Labels:     []string{"/skills/AI"},
	}

	tests := []struct {
		name    string
		filters []Filter
		want    bool
	}{
		{
			name: "matches all filters",
			filters: []Filter{
				EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED),
				CIDFilter("bafytest123"),
				LabelFilter("/skills/AI"),
			},
			want: true,
		},
		{
			name: "fails one filter",
			filters: []Filter{
				EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED),
				CIDFilter("different-cid"),
			},
			want: false,
		},
		{
			name:    "empty filters matches everything",
			filters: []Filter{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Matches(event, tt.filters); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComplexFilterComposition(t *testing.T) {
	// Complex filter: (PUSHED OR PUBLISHED) AND AI labels AND NOT deleted
	filter := And(
		Or(
			EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED),
			EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED),
		),
		LabelFilter("/skills/AI"),
		Not(EventTypeFilter(eventsv1.EventType_EVENT_TYPE_RECORD_DELETED)),
	)

	tests := []struct {
		name  string
		event *Event
		want  bool
	}{
		{
			name: "matches: pushed + AI",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Labels: []string{"/skills/AI"},
			},
			want: true,
		},
		{
			name: "matches: published + AI",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
				Labels: []string{"/skills/AI"},
			},
			want: true,
		},
		{
			name: "fails: deleted (even with AI)",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_DELETED,
				Labels: []string{"/skills/AI"},
			},
			want: false,
		},
		{
			name: "fails: pushed but no AI",
			event: &Event{
				Type:   eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				Labels: []string{"/domains/medical"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.event.ID = TestEventID
			tt.event.Timestamp = time.Now()
			tt.event.ResourceID = TestCID123

			if got := filter(tt.event); got != tt.want {
				t.Errorf("Complex filter = %v, want %v", got, tt.want)
			}
		})
	}
}
