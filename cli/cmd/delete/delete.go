// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:predeclared,wrapcheck
package delete

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	cidutil "github.com/agntcy/dir/cli/util/cids"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

func init() {
	presenter.AddOutputFlags(Command)
	Command.Flags().BoolVar(&deleteOpts.FromStdin, "stdin", false,
		"Read CIDs from standard input. Supports JSON array output from 'dirctl search --output json' and line-delimited CIDs.")
}

var deleteOpts struct {
	FromStdin bool
}

var Command = &cobra.Command{
	Use:   "delete <cid> [cid...]",
	Short: "Delete records from Directory store",
	Long: `This command deletes records from the Directory store.

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
	Args: func(cmd *cobra.Command, args []string) error {
		if deleteOpts.FromStdin {
			return cobra.MaximumNArgs(0)(cmd, args)
		}
		return cobra.MinimumNArgs(1)(cmd, args)
	},
	RunE: runCommand,
}

func runCommand(cmd *cobra.Command, args []string) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	cids := append([]string{}, args...)
	if deleteOpts.FromStdin {
		stdinCIDs, err := cidutil.ReadFrom(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("failed to read CIDs from stdin: %w", err)
		}
		cids = append(cids, stdinCIDs...)
	}
	cids = cidutil.Deduplicate(cids)
	if len(cids) == 0 {
		return errors.New("at least one CID is required (pass arguments or use --stdin)")
	}

	recordRefs := make([]*corev1.RecordRef, 0, len(cids))
	for _, cid := range cids {
		recordRefs = append(recordRefs, &corev1.RecordRef{Cid: cid})
	}
	if err := c.DeleteBatch(cmd.Context(), recordRefs); err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}

	result := map[string]any{
		"count":  len(cids),
		"cids":   cids,
		"status": "Successfully deleted records",
	}
	if len(cids) == 1 {
		result["cid"] = cids[0]
	}
	return presenter.PrintMessage(cmd, "Delete", "Successfully deleted record(s)", result)
}
