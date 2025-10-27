// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "events",
	Short: "Stream real-time system events",
	Long: `Stream real-time events from the Directory system.

This command allows you to monitor system activity by subscribing to
events from various services (store, routing, sync, signing).

Examples:

1. Listen to all events:
   dirctl events listen

2. Filter by event type:
   dirctl events listen --types RECORD_PUSHED,RECORD_PUBLISHED

3. Filter by labels:
   dirctl events listen --labels /skills/AI

4. Output formats:
   dirctl events listen --output jsonl    # Streaming JSON (one per line)
   dirctl events listen --output json     # Pretty-printed JSON
   dirctl events listen --output raw      # Resource IDs only

Events are delivered from subscription time forward (no history).
The stream remains active until interrupted (Ctrl+C).
`,
}

func init() {
	// Add subcommands
	Command.AddCommand(listenCmd)

	// Add output format flags
	presenter.AddOutputFlags(listenCmd)
}
