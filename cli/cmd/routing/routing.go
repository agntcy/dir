// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "routing",
	Short: "Routing operations for record discovery and announcement",
	Long: `Routing operations for record discovery and announcement.

This command group provides access to all routing-specific operations:

- publish: Announce records to the network for discovery
- unpublish: Remove records from network discovery
- list: Query local records with filtering
- search: Discover remote records from other peers
- info: Show routing statistics and summary information

Examples:

1. Publish a record to the network:
   dirctl routing publish <cid>

2. List local records with skill filter:
   dirctl routing list --skill "AI"

3. Search for remote records across the network:
   dirctl routing search --skill "AI" --limit 10

4. Unpublish a record from the network:
   dirctl routing unpublish <cid>

This follows clear service separation - all routing API operations are grouped together.
`,
}

func init() {
	// Add all routing subcommands
	Command.AddCommand(publishCmd)
	Command.AddCommand(unpublishCmd)
	Command.AddCommand(listCmd)
	Command.AddCommand(searchCmd)
	Command.AddCommand(infoCmd)

	// Add output format flags to routing subcommands
	presenter.AddOutputFlags(publishCmd)
	presenter.AddOutputFlags(unpublishCmd)
}
