// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:predeclared
package delete

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "delete",
	Short: "Delete agent model from Directory store",
	Long: `This command deletes an agent model from the Directory store.

Usage example:

	dirctl delete <digest>

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("digest is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, digest string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Delete object from store
	err := c.Delete(cmd.Context(), &corev1.RecordRef{
		Cid: digest, // Use digest as CID directly
	})
	if err != nil {
		return fmt.Errorf("failed to delete agent model: %w", err)
	}

	presenter.Printf(cmd, "Deleted agent model with digest: %s\n", digest)

	return nil
}
