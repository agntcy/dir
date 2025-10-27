// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen to real-time system events",
	Long: `Listen to real-time system events with optional filtering.

Events are streamed from the Directory server in real-time.
Only events occurring after subscription are delivered (no history).
The stream remains active until interrupted (Ctrl+C).

Examples:

1. Listen to all events:
   dirctl events listen

2. Filter by specific event types:
   dirctl events listen --types RECORD_PUSHED,RECORD_PUBLISHED

3. Filter by labels (AI-related records):
   dirctl events listen --labels /skills/AI

4. Filter by CID:
   dirctl events listen --cids bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi

5. Combine filters:
   dirctl events listen --types RECORD_PUSHED --labels /skills/AI --output jsonl

Available event types:
- Store: RECORD_PUSHED, RECORD_PULLED, RECORD_DELETED
- Routing: RECORD_PUBLISHED, RECORD_UNPUBLISHED
- Sync: SYNC_CREATED, SYNC_COMPLETED, SYNC_FAILED
- Sign: RECORD_SIGNED
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runListenCommand(cmd)
	},
}

// Listen command options.
var listenOpts struct {
	EventTypes   []string
	LabelFilters []string
	CIDFilters   []string
}

func init() {
	listenCmd.Flags().StringArrayVar(&listenOpts.EventTypes, "types", nil,
		"Event types to filter (e.g., --types RECORD_PUSHED,RECORD_PUBLISHED)")
	listenCmd.Flags().StringArrayVar(&listenOpts.LabelFilters, "labels", nil,
		"Label filters (e.g., --labels /skills/AI --labels /domains/research)")
	listenCmd.Flags().StringArrayVar(&listenOpts.CIDFilters, "cids", nil,
		"CID filters (e.g., --cids bafyxxx)")
}

func runListenCommand(cmd *cobra.Command) error {
	// Get client from context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Parse event types from strings to enums
	eventTypes, err := parseEventTypes(listenOpts.EventTypes)
	if err != nil {
		return fmt.Errorf("invalid event types: %w", err)
	}

	// Build request
	req := &eventsv1.ListenRequest{
		EventTypes:   eventTypes,
		LabelFilters: listenOpts.LabelFilters,
		CidFilters:   listenOpts.CIDFilters,
	}

	// Start listening
	result, err := c.ListenStream(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to start event stream: %w", err)
	}

	// Show metadata only in human format (route to stderr for structured formats)
	opts := presenter.GetOutputOptions(cmd)
	if opts.Format == presenter.FormatHuman {
		presenter.Printf(cmd, "Listening to events (press Ctrl+C to stop)...\n")

		if len(eventTypes) > 0 {
			presenter.Printf(cmd, "Event types: %v\n", listenOpts.EventTypes)
		}

		if len(listenOpts.LabelFilters) > 0 {
			presenter.Printf(cmd, "Label filters: %v\n", listenOpts.LabelFilters)
		}

		if len(listenOpts.CIDFilters) > 0 {
			presenter.Printf(cmd, "CID filters: %v\n", listenOpts.CIDFilters)
		}

		presenter.Printf(cmd, "\n")
	}

	// Stream events using StreamResult pattern
	for {
		select {
		case resp := <-result.ResCh():
			event := resp.GetEvent()
			if event != nil {
				displayEvent(cmd, event)
			}
		case err := <-result.ErrCh():
			return fmt.Errorf("error receiving event: %w", err)
		case <-result.DoneCh():
			// Stream ended normally
			return nil
		case <-cmd.Context().Done():
			// Return unwrapped context error so callers can check for context.Canceled
			//nolint:wrapcheck
			return cmd.Context().Err()
		}
	}
}

// displayEvent formats and displays an event.
func displayEvent(cmd *cobra.Command, event *eventsv1.Event) {
	// Get output options
	opts := presenter.GetOutputOptions(cmd)

	switch opts.Format {
	case presenter.FormatJSON:
		// Pretty-printed JSON
		data, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			presenter.Errorf(cmd, "Error marshaling event: %v\n", err)

			return
		}

		presenter.Printf(cmd, "%s\n", string(data))

	case presenter.FormatJSONL:
		// Compact JSON for streaming (newline-delimited)
		data, err := json.Marshal(event)
		if err != nil {
			presenter.Errorf(cmd, "Error marshaling event: %v\n", err)

			return
		}

		presenter.Printf(cmd, "%s\n", string(data))

	case presenter.FormatRaw:
		// Just print resource ID
		presenter.Printf(cmd, "%s\n", event.GetResourceId())

	case presenter.FormatHuman:
		// Human-readable format
		eventType := strings.TrimPrefix(event.GetType().String(), "EVENT_TYPE_")

		presenter.Printf(cmd, "[%s] %s: %s",
			event.GetTimestamp().AsTime().Format("15:04:05"),
			eventType,
			event.GetResourceId())

		if len(event.GetLabels()) > 0 {
			presenter.Printf(cmd, " (labels: %s)", strings.Join(event.GetLabels(), ", "))
		}

		if len(event.GetMetadata()) > 0 {
			presenter.Printf(cmd, " %v", event.GetMetadata())
		}

		presenter.Printf(cmd, "\n")
	}
}

// parseEventTypes converts string event type names to enum values.
func parseEventTypes(typeStrings []string) ([]eventsv1.EventType, error) {
	if len(typeStrings) == 0 {
		return nil, nil
	}

	var eventTypes []eventsv1.EventType

	for _, typeStr := range typeStrings {
		// Handle comma-separated values
		parts := strings.Split(typeStr, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Add EVENT_TYPE_ prefix if not present
			if !strings.HasPrefix(part, "EVENT_TYPE_") {
				part = "EVENT_TYPE_" + part
			}

			// Parse enum value
			enumValue, ok := eventsv1.EventType_value[part]
			if !ok {
				return nil, fmt.Errorf("unknown event type: %s (use one of: RECORD_PUSHED, RECORD_PULLED, etc.)", part)
			}

			eventTypes = append(eventTypes, eventsv1.EventType(enumValue))
		}
	}

	return eventTypes, nil
}
