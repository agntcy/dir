// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"strings"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
)

// Filter is a function that checks if an event matches certain criteria.
// Filters are composable and can be combined using And, Or, and Not operators.
type Filter func(*Event) bool

// BuildFilters converts a ListenRequest into a list of filter functions.
// These filters are applied when determining which events to deliver to a subscriber.
//
// If no filters are specified in the request, returns an empty slice (matches all events).
func BuildFilters(req *eventsv1.ListenRequest) []Filter {
	var filters []Filter

	if len(req.GetEventTypes()) > 0 {
		filters = append(filters, EventTypeFilter(req.GetEventTypes()...))
	}

	if len(req.GetCidFilters()) > 0 {
		filters = append(filters, CIDFilter(req.GetCidFilters()...))
	}

	if len(req.GetLabelFilters()) > 0 {
		filters = append(filters, LabelFilter(req.GetLabelFilters()...))
	}

	return filters
}

// Matches checks if an event passes all the given filters.
// Returns true if all filters pass (AND logic), false otherwise.
// If filters slice is empty, returns true (matches everything).
func Matches(event *Event, filters []Filter) bool {
	for _, filter := range filters {
		if !filter(event) {
			return false
		}
	}

	return true
}

// EventTypeFilter creates a filter that matches events with any of the specified types.
// Returns true if the event's type matches any of the provided types (OR logic).
//
// Example:
//
//	filter := EventTypeFilter(
//	    eventsv1.EVENT_TYPE_RECORD_PUSHED,
//	    eventsv1.EVENT_TYPE_RECORD_PUBLISHED,
//	)
func EventTypeFilter(types ...eventsv1.EventType) Filter {
	return func(e *Event) bool {
		for _, t := range types {
			if e.Type == t {
				return true
			}
		}

		return false
	}
}

// CIDFilter creates a filter that matches events with any of the specified CIDs.
// Returns true if the event's resource ID matches any of the provided CIDs (OR logic).
//
// Example:
//
//	filter := CIDFilter("bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi")
func CIDFilter(cids ...string) Filter {
	return func(e *Event) bool {
		for _, cid := range cids {
			if e.ResourceID == cid {
				return true
			}
		}

		return false
	}
}

// LabelFilter creates a filter that matches events with labels containing any of the specified substrings.
// Returns true if any of the event's labels contains any of the filter strings (OR logic).
// Uses substring matching for flexibility.
//
// Example:
//
//	filter := LabelFilter("/skills/AI", "/domains/research")
//	// Matches events with labels like "/skills/AI/ML" or "/domains/research/quantum"
func LabelFilter(labelFilters ...string) Filter {
	return func(e *Event) bool {
		for _, filter := range labelFilters {
			for _, label := range e.Labels {
				if strings.Contains(label, filter) {
					return true
				}
			}
		}

		return false
	}
}

// Or combines multiple filters with OR logic.
// Returns true if ANY of the filters matches (short-circuits on first match).
//
// Example:
//
//	filter := Or(
//	    EventTypeFilter(EVENT_TYPE_RECORD_PUSHED),
//	    EventTypeFilter(EVENT_TYPE_RECORD_PULLED),
//	)
func Or(filters ...Filter) Filter {
	return func(e *Event) bool {
		for _, filter := range filters {
			if filter(e) {
				return true
			}
		}

		return false
	}
}

// And combines multiple filters with AND logic.
// Returns true if ALL of the filters match (short-circuits on first failure).
//
// Example:
//
//	filter := And(
//	    EventTypeFilter(EVENT_TYPE_RECORD_PUSHED),
//	    LabelFilter("/skills/AI"),
//	)
func And(filters ...Filter) Filter {
	return func(e *Event) bool {
		for _, filter := range filters {
			if !filter(e) {
				return false
			}
		}

		return true
	}
}

// Not negates a filter.
// Returns true if the filter does NOT match, false if it does match.
//
// Example:
//
//	filter := Not(EventTypeFilter(EVENT_TYPE_RECORD_DELETED))
//	// Matches all events EXCEPT deletions
func Not(filter Filter) Filter {
	return func(e *Event) bool {
		return !filter(e)
	}
}
