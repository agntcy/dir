// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:predeclared,wrapcheck
package delete

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	cidsUtils "github.com/agntcy/dir/cli/util/cids"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

func init() {
	// Add output format flags
	presenter.AddOutputFlags(Command)

	Command.Flags().BoolVar(&opts.FromStdin, "stdin", false, cidsUtils.StdinFlagUsage)
}

var opts struct {
	FromStdin bool
}

var Command = &cobra.Command{
	Use:   "delete <cid> [cid...]",
	Short: "Delete records from Directory store",
	Long: `This command deletes one or more records from the Directory store.
Records submitted together are deleted over a single stream.

Usage examples:

1. Delete a record:

	dirctl delete <cid>

2. Delete multiple records in a single request:

	dirctl delete <cid1> <cid2> <cid3>

3. Delete records from stdin (JSON array or line-delimited CIDs):

	dirctl search --format cid --limit 100 --output json | dirctl delete --stdin

4. Output formats:

	# Delete with JSON confirmation
	dirctl delete <cid> --output json
	
	# Delete with raw output for scripting
	dirctl delete <cid> --output raw

`,
	Args: cidsUtils.Args(&opts.FromStdin),
	RunE: runCommand,
}

func runCommand(cmd *cobra.Command, args []string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	cids, err := cidsUtils.Collect(args, cmd.InOrStdin(), opts.FromStdin)
	if err != nil {
		return err
	}

	recordRefs := make([]*corev1.RecordRef, 0, len(cids))
	for _, cid := range cids {
		recordRefs = append(recordRefs, &corev1.RecordRef{Cid: cid})
	}

	// Delete objects from store over a single stream
	if err := c.DeleteBatch(cmd.Context(), recordRefs); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	// Output in the appropriate format
	if len(cids) == 1 {
		return presenter.PrintMessage(cmd, "record", "Deleted record with CID", cids[0])
	}

	return presenter.PrintMessage(cmd, "records", "Deleted records with CIDs", cids)
}
