// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"bytes"
	"strings"
	"testing"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestParseEventTypes_Empty tests parsing empty input.
func TestParseEventTypes_Empty(t *testing.T) {
	result, err := parseEventTypes(nil)
	require.NoError(t, err)
	assert.Nil(t, result)

	result, err = parseEventTypes([]string{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseEventTypes_SingleType tests parsing a single event type.
func TestParseEventTypes_SingleType(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []eventsv1.EventType
	}{
		{
			name:     "without prefix",
			input:    []string{"RECORD_PUSHED"},
			expected: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
		},
		{
			name:     "with prefix",
			input:    []string{"EVENT_TYPE_RECORD_PUSHED"},
			expected: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
		},
		{
			name:     "with whitespace",
			input:    []string{"  RECORD_PUSHED  "},
			expected: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEventTypes(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseEventTypes_MultipleTypes tests parsing multiple event types.
func TestParseEventTypes_MultipleTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []eventsv1.EventType
	}{
		{
			name:  "comma-separated in single string",
			input: []string{"RECORD_PUSHED,RECORD_PULLED"},
			expected: []eventsv1.EventType{
				eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				eventsv1.EventType_EVENT_TYPE_RECORD_PULLED,
			},
		},
		{
			name:  "separate strings",
			input: []string{"RECORD_PUSHED", "RECORD_PULLED"},
			expected: []eventsv1.EventType{
				eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				eventsv1.EventType_EVENT_TYPE_RECORD_PULLED,
			},
		},
		{
			name:  "mixed with whitespace",
			input: []string{"RECORD_PUSHED , RECORD_PULLED "},
			expected: []eventsv1.EventType{
				eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				eventsv1.EventType_EVENT_TYPE_RECORD_PULLED,
			},
		},
		{
			name:  "all store event types",
			input: []string{"RECORD_PUSHED,RECORD_PULLED,RECORD_DELETED"},
			expected: []eventsv1.EventType{
				eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				eventsv1.EventType_EVENT_TYPE_RECORD_PULLED,
				eventsv1.EventType_EVENT_TYPE_RECORD_DELETED,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEventTypes(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseEventTypes_AllEventTypes tests all valid event types.
func TestParseEventTypes_AllEventTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected eventsv1.EventType
	}{
		{"RECORD_PUSHED", eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
		{"RECORD_PULLED", eventsv1.EventType_EVENT_TYPE_RECORD_PULLED},
		{"RECORD_DELETED", eventsv1.EventType_EVENT_TYPE_RECORD_DELETED},
		{"RECORD_PUBLISHED", eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED},
		{"RECORD_UNPUBLISHED", eventsv1.EventType_EVENT_TYPE_RECORD_UNPUBLISHED},
		{"SYNC_CREATED", eventsv1.EventType_EVENT_TYPE_SYNC_CREATED},
		{"SYNC_COMPLETED", eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED},
		{"SYNC_FAILED", eventsv1.EventType_EVENT_TYPE_SYNC_FAILED},
		{"RECORD_SIGNED", eventsv1.EventType_EVENT_TYPE_RECORD_SIGNED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseEventTypes([]string{tt.input})
			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.Equal(t, tt.expected, result[0])
		})
	}
}

// TestParseEventTypes_EmptyStrings tests handling of empty strings.
func TestParseEventTypes_EmptyStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []eventsv1.EventType
	}{
		{
			name:     "single empty string",
			input:    []string{""},
			expected: nil,
		},
		{
			name:     "empty string in comma list",
			input:    []string{"RECORD_PUSHED,,RECORD_PULLED"},
			expected: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, eventsv1.EventType_EVENT_TYPE_RECORD_PULLED},
		},
		{
			name:     "whitespace only",
			input:    []string{"   "},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEventTypes(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseEventTypes_InvalidType tests error handling for invalid event types.
func TestParseEventTypes_InvalidType(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		errContains string
	}{
		{
			name:        "completely invalid",
			input:       []string{"INVALID_TYPE"},
			errContains: "unknown event type",
		},
		{
			name:        "typo in type",
			input:       []string{"RECORD_PUSHD"},
			errContains: "unknown event type",
		},
		{
			name:        "invalid in list",
			input:       []string{"RECORD_PUSHED,INVALID_TYPE"},
			errContains: "unknown event type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEventTypes(tt.input)
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

// createTestEvent creates a test event for display testing.
func createTestEvent(resourceID string, eventType eventsv1.EventType, labels []string, metadata map[string]string) *eventsv1.Event {
	return &eventsv1.Event{
		Id:         "test-event-id",
		Type:       eventType,
		ResourceId: resourceID,
		Timestamp:  timestamppb.Now(),
		Labels:     labels,
		Metadata:   metadata,
	}
}

// TestDisplayEvent_FormatJSON tests JSON format output.
func TestDisplayEvent_FormatJSON(t *testing.T) {
	event := createTestEvent("test-cid", eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, nil, nil)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.Flags().String("output", "json", "")
	require.NoError(t, cmd.Flags().Set("output", "json"))

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	displayEvent(cmd, event)

	output := stdout.String()
	assert.Contains(t, output, `"resource_id": "test-cid"`)
	assert.Contains(t, output, `"id": "test-event-id"`)
	// Pretty-printed JSON should have indentation
	assert.Contains(t, output, "  ")
	assert.Contains(t, output, `"type"`)
}

// TestDisplayEvent_FormatJSONL tests JSONL format output.
func TestDisplayEvent_FormatJSONL(t *testing.T) {
	event := createTestEvent("test-cid", eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, nil, nil)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.Flags().String("output", "jsonl", "")
	require.NoError(t, cmd.Flags().Set("output", "jsonl"))

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	displayEvent(cmd, event)

	output := stdout.String()
	assert.Contains(t, output, `"resource_id":"test-cid"`)
	assert.Contains(t, output, `"id":"test-event-id"`)
	assert.Contains(t, output, `"type"`)
	// JSONL should be compact (no extra whitespace)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 1, "JSONL should be single line")
}

// TestDisplayEvent_FormatRaw tests raw format output.
func TestDisplayEvent_FormatRaw(t *testing.T) {
	event := createTestEvent("test-cid-123", eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, nil, nil)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.Flags().String("output", "raw", "")
	require.NoError(t, cmd.Flags().Set("output", "raw"))

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	displayEvent(cmd, event)

	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "test-cid-123", output)
}

// TestDisplayEvent_FormatHuman tests human-readable format.
func TestDisplayEvent_FormatHuman(t *testing.T) {
	tests := []struct {
		name     string
		event    *eventsv1.Event
		contains []string
	}{
		{
			name:  "basic event",
			event: createTestEvent("test-cid", eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, nil, nil),
			contains: []string{
				"RECORD_PUSHED",
				"test-cid",
			},
		},
		{
			name:  "event with labels",
			event: createTestEvent("test-cid", eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, []string{"/skills/AI", "/domains/research"}, nil),
			contains: []string{
				"RECORD_PUSHED",
				"test-cid",
				"labels: /skills/AI, /domains/research",
			},
		},
		{
			name:  "event with metadata",
			event: createTestEvent("test-cid", eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, nil, map[string]string{"key": "value"}),
			contains: []string{
				"RECORD_PUSHED",
				"test-cid",
				"key:value",
			},
		},
		{
			name: "event with labels and metadata",
			event: createTestEvent(
				"test-cid",
				eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				[]string{"/skills/AI"},
				map[string]string{"key": "value"},
			),
			contains: []string{
				"RECORD_PUSHED",
				"test-cid",
				"labels: /skills/AI",
				"key:value",
			},
		},
		{
			name:  "different event type",
			event: createTestEvent("sync-id", eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED, nil, nil),
			contains: []string{
				"SYNC_COMPLETED",
				"sync-id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.Flags().String("output", "human", "")
			require.NoError(t, cmd.Flags().Set("output", "human"))

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			displayEvent(cmd, tt.event)

			output := stdout.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
			// Human format should include timestamp
			assert.Contains(t, output, ":")
		})
	}
}

// TestDisplayEvent_AllFormats tests that all formats handle the same event.
func TestDisplayEvent_AllFormats(t *testing.T) {
	event := createTestEvent(
		"test-cid-complete",
		eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
		[]string{"/skills/ML"},
		map[string]string{"source": "test"},
	)

	formats := []struct {
		format string
		check  func(t *testing.T, output string)
	}{
		{
			format: "json",
			check: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "test-cid-complete")
				assert.Contains(t, output, `"type"`)
				assert.Contains(t, output, "/skills/ML")
			},
		},
		{
			format: "jsonl",
			check: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "test-cid-complete")
				lines := strings.Split(strings.TrimSpace(output), "\n")
				assert.Len(t, lines, 1)
			},
		},
		{
			format: "raw",
			check: func(t *testing.T, output string) {
				t.Helper()
				assert.Equal(t, "test-cid-complete", strings.TrimSpace(output))
			},
		},
		{
			format: "human",
			check: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "test-cid-complete")
				assert.Contains(t, output, "RECORD_PUBLISHED")
				assert.Contains(t, output, "/skills/ML")
			},
		},
	}

	for _, tt := range formats {
		t.Run(tt.format, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.Flags().String("output", tt.format, "")
			require.NoError(t, cmd.Flags().Set("output", tt.format))

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			displayEvent(cmd, event)

			output := stdout.String()
			assert.NotEmpty(t, output)
			tt.check(t, output)
		})
	}
}

// TestDisplayEvent_DefaultFormat tests default format is human.
func TestDisplayEvent_DefaultFormat(t *testing.T) {
	event := createTestEvent("test-cid", eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, nil, nil)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	// Don't set output flag - should default to human

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	displayEvent(cmd, event)

	output := stdout.String()
	// Human format should contain event type without EVENT_TYPE_ prefix
	assert.Contains(t, output, "RECORD_PUSHED")
	assert.Contains(t, output, "test-cid")
	// Should have timestamp format
	assert.Contains(t, output, ":")
}

// TestListenCmd_Initialization tests that listenCmd is properly initialized.
func TestListenCmd_Initialization(t *testing.T) {
	assert.NotNil(t, listenCmd)
	assert.Equal(t, "listen", listenCmd.Use)
	assert.NotEmpty(t, listenCmd.Short)
	assert.NotEmpty(t, listenCmd.Long)
	assert.NotNil(t, listenCmd.RunE)

	// Check flags are registered
	typesFlag := listenCmd.Flags().Lookup("types")
	assert.NotNil(t, typesFlag)

	labelsFlag := listenCmd.Flags().Lookup("labels")
	assert.NotNil(t, labelsFlag)

	cidsFlag := listenCmd.Flags().Lookup("cids")
	assert.NotNil(t, cidsFlag)
}

// TestListenOpts_Structure tests the listenOpts structure.
func TestListenOpts_Structure(t *testing.T) {
	// Reset to ensure clean state
	listenOpts.EventTypes = []string{"RECORD_PUSHED"}
	listenOpts.LabelFilters = []string{"/skills/AI"}
	listenOpts.CIDFilters = []string{"test-cid"}

	assert.Equal(t, []string{"RECORD_PUSHED"}, listenOpts.EventTypes)
	assert.Equal(t, []string{"/skills/AI"}, listenOpts.LabelFilters)
	assert.Equal(t, []string{"test-cid"}, listenOpts.CIDFilters)

	// Reset after test
	listenOpts.EventTypes = nil
	listenOpts.LabelFilters = nil
	listenOpts.CIDFilters = nil
}

// TestDisplayEvent_ErrorHandling tests error handling in displayEvent.
func TestDisplayEvent_ErrorHandling(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"json error handling", "json"},
		{"jsonl error handling", "jsonl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create event (even with nil, the function should handle it gracefully)
			event := &eventsv1.Event{
				Id:         "test",
				Type:       eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
				ResourceId: "test-cid",
				Timestamp:  timestamppb.Now(),
			}

			cmd := &cobra.Command{}
			cmd.SetOut(&bytes.Buffer{})

			var stderr bytes.Buffer
			cmd.SetErr(&stderr)
			cmd.Flags().String("output", tt.format, "")
			require.NoError(t, cmd.Flags().Set("output", tt.format))

			// Should not panic
			displayEvent(cmd, event)
		})
	}
}

// TestDisplayEvent_HumanWithNoLabelsOrMetadata tests human format with minimal event.
func TestDisplayEvent_HumanWithNoLabelsOrMetadata(t *testing.T) {
	event := createTestEvent("minimal-cid", eventsv1.EventType_EVENT_TYPE_RECORD_DELETED, nil, nil)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.Flags().String("output", "human", "")
	require.NoError(t, cmd.Flags().Set("output", "human"))

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	displayEvent(cmd, event)

	output := stdout.String()
	assert.Contains(t, output, "RECORD_DELETED")
	assert.Contains(t, output, "minimal-cid")
	// Should not contain labels or metadata sections
	assert.NotContains(t, output, "labels:")
}

// TestDisplayEvent_AllEventTypes tests display for all event types.
func TestDisplayEvent_AllEventTypes(t *testing.T) {
	eventTypes := []struct {
		eventType eventsv1.EventType
		name      string
	}{
		{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED, "RECORD_PUSHED"},
		{eventsv1.EventType_EVENT_TYPE_RECORD_PULLED, "RECORD_PULLED"},
		{eventsv1.EventType_EVENT_TYPE_RECORD_DELETED, "RECORD_DELETED"},
		{eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED, "RECORD_PUBLISHED"},
		{eventsv1.EventType_EVENT_TYPE_RECORD_UNPUBLISHED, "RECORD_UNPUBLISHED"},
		{eventsv1.EventType_EVENT_TYPE_SYNC_CREATED, "SYNC_CREATED"},
		{eventsv1.EventType_EVENT_TYPE_SYNC_COMPLETED, "SYNC_COMPLETED"},
		{eventsv1.EventType_EVENT_TYPE_SYNC_FAILED, "SYNC_FAILED"},
		{eventsv1.EventType_EVENT_TYPE_RECORD_SIGNED, "RECORD_SIGNED"},
	}

	for _, tt := range eventTypes {
		t.Run(tt.name, func(t *testing.T) {
			event := createTestEvent("test-cid", tt.eventType, nil, nil)

			cmd := &cobra.Command{}
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.Flags().String("output", "human", "")
			require.NoError(t, cmd.Flags().Set("output", "human"))

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			displayEvent(cmd, event)

			output := stdout.String()
			assert.Contains(t, output, tt.name)
			assert.Contains(t, output, "test-cid")
		})
	}
}
