// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// Command is the parent command for naming operations.
var Command = &cobra.Command{
	Use:   "naming",
	Short: "Domain naming and verification operations",
	Long: `Domain naming and verification operations.

This command group provides access to domain ownership verification:

- verify: Verify domain ownership for a signed record
- check: Check if a record has verified domain ownership
- list: List all verified agents for a domain

Domain verification proves that a record's signing key is authorized by the
domain claimed in the record's name. This enables trustworthy human-readable
naming (e.g., "cisco.com/marketing-agent").

Verification methods:
1. DNS TXT record: _oasf.<domain> with format "v=oasf1; k=<type>; p=<key>"
2. Well-known file: https://<domain>/.well-known/oasf.json

Examples:

1. Verify domain ownership after signing:
   dirctl naming verify <cid>

2. Check if a record has verified domain ownership:
   dirctl naming check <cid>

3. List all verified agents for a domain:
   dirctl naming list cisco.com
`,
}

func init() {
	// Add all naming subcommands
	Command.AddCommand(verifyCmd)
	Command.AddCommand(checkCmd)
	Command.AddCommand(listCmd)

	// Add output format flags to naming subcommands
	presenter.AddOutputFlags(verifyCmd)
	presenter.AddOutputFlags(checkCmd)
	presenter.AddOutputFlags(listCmd)
}
