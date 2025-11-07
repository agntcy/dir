// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:predeclared,wrapcheck
package delete

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
	Use:   "delete",
	Short: "Delete record from Directory store",
	Long: `This command deletes a record from the Directory store.

Usage examples:

1. Delete a record:

	dirctl delete <cid>

2. Output formats:

	# Delete with JSON confirmation
	dirctl delete <cid> --output json
	
	# Delete with raw output for scripting
	dirctl delete <cid> --output raw

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("cid is a required argument")
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

	// Delete object from store
	err := c.Delete(cmd.Context(), &corev1.RecordRef{
		Cid: cid,
	})
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	// Output in the appropriate format
	return presenter.PrintMessage(cmd, "record", "Deleted record with CID", cid)
}
