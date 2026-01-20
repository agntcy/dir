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
	Short: "Name verification operations",
	Long: `Name verification operations.

This command group provides access to name ownership verification:

- verify: Check if a record has verified name ownership

Name verification proves that a record's signing key is authorized by the
domain claimed in the record's name. This enables trustworthy human-readable
naming (e.g., "https://cisco.com/marketing-agent").

Verification is performed automatically when signing a record with 'dirctl sign'.
Use 'dirctl naming verify' to check the verification status.

Protocol prefixes (required for verification):
- dns://domain/path - verify using DNS TXT records
- https://domain/path - verify using JWKS well-known file (RFC 7517)
- http://domain/path - verify using JWKS via HTTP (testing only)

Records without a protocol prefix will not be verified.

Verification methods:
1. DNS TXT record: _dir_nsys.<domain> with format "schema=v1; v=pubkey; k=<type>; p=<key>"
2. JWKS well-known file: <scheme>://<domain>/.well-known/jwks.json

Examples:

1. Check name verification status:
   dirctl naming verify <cid>

2. Check with JSON output:
   dirctl naming verify <cid> --output json
`,
}

func init() {
	// Add all naming subcommands
	Command.AddCommand(verifyCmd)

	// Add output format flags to naming subcommands
	presenter.AddOutputFlags(verifyCmd)
}
