// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package info

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

func init() {
	// Add output format flags
	presenter.AddOutputFlags(Command)
}

var Command = &cobra.Command{
	Use:   "info <cid-or-name[:version][@digest]>",
	Short: "Check info about an object in Directory store",
	Long: `Lookup and get basic metadata about an object pushed to the Directory store.

You can lookup by CID or by name. The command auto-detects whether the argument is a CID or a name:
- If it's a valid CID (e.g., bafyrei...), it looks up directly by CID
- Otherwise, it resolves the name to a CID and looks it up

When looking up by name without a version, the latest version (by semver) is used.

Usage examples:

1. Get info by CID:

	dirctl info bafyreib...

2. Get info by name (latest version):

	dirctl info cisco.com/marketing-agent

3. Get info by name with specific version:

	dirctl info cisco.com/marketing-agent:v1.0.0

4. Get info with hash verification:

	dirctl info cisco.com/marketing-agent@bafyreib...

5. Output formats:

	# Get info as JSON
	dirctl info <cid-or-name> --output json
	
	# Get raw info data
	dirctl info <cid-or-name> --output raw

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("cid or name is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, input string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Resolve the input to a CID
	recordCID, err := reference.ResolveToCID(cmd.Context(), c, input)
	if err != nil {
		return err
	}

	// Fetch info from store
	info, err := c.Lookup(cmd.Context(), &corev1.RecordRef{
		Cid: recordCID,
	})
	if err != nil {
		return fmt.Errorf("failed to lookup data: %w", err)
	}

	// Output in the appropriate format
	return presenter.PrintMessage(cmd, "info", "Record information", info)
}
