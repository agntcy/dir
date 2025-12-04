// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "search",
	Short: "Search for records in the directory",
	Long: `Search for records in the directory using various filters and options.

This command group provides access to search operations:

- cids: Search and return only record CIDs (efficient for piping)
- records: Search and return full record data

Examples:

1. Search for CIDs only (efficient for piping):
   dirctl search cids --name "web*" | xargs -I {} dirctl pull {}

2. Search and get full records:
   dirctl search records --name "web*" --output json

3. Wildcard search examples:
   dirctl search cids --name "web*"
   dirctl search records --version "v1.*"
   dirctl search cids --skill "python*" --skill "*script"
   dirctl search records --domain "*education*"

4. Comparison operators (for version, created-at, schema-version):
   dirctl search records --version ">=1.0.0" --version "<2.0.0"
   dirctl search cids --created-at ">=2024-01-01"

Supported wildcards:
  * - matches zero or more characters
  ? - matches exactly one character
  [] - matches any character within brackets (e.g., [0-9], [a-z])
`,
}

func init() {
	// Add all search subcommands
	Command.AddCommand(cidsCmd)
	Command.AddCommand(recordsCmd)

	// Add output format flags to search subcommands
	presenter.AddOutputFlags(cidsCmd)
	presenter.AddOutputFlags(recordsCmd)
}
