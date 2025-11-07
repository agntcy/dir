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
	"github.com/spf13/cobra"
)

func init() {
	// Add output format flags
	presenter.AddOutputFlags(Command)
}

var Command = &cobra.Command{
	Use:   "info",
	Short: "Check info about an object in Directory store",
	Long: `Lookup and get basic metadata about an object pushed to the Directory store.

Usage examples:

1. Get info about a record:

	dirctl info <cid>

2. Output formats:

	# Get info as JSON
	dirctl info <cid> --output json
	
	# Get raw info data
	dirctl info <cid> --output raw

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("exactly one argument is required which is the cid of the object")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, cid string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Fetch info from store
	info, err := c.Lookup(cmd.Context(), &corev1.RecordRef{
		Cid: cid,
	})
	if err != nil {
		return fmt.Errorf("failed to pull data: %w", err)
	}

	// Output in the appropriate format
	return presenter.PrintMessage(cmd, "info", "Record information", info)
}
