// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "publish",
	Short: "Publish compiled agent model to DHT allowing content discovery",
	Long: `Usage example:

	# Publish the data across the network.
  	# It is not guaranteed that this will succeed.
  	dirctl publish <digest>

   	# Publish the data only to the local routing table.
    dirctl publish <digest> --local

`,
	RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
		return runCommand(cmd, args)
	},
}

func runCommand(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("requires exactly 1 argument")
	}

	digest := args[0]

	// Get the client from the context.
	c, ok := util.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Lookup from digest
	meta, err := c.Lookup(cmd.Context(), &coretypes.ObjectRef{
		Digest: digest,
	})
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	presenter.Printf(cmd, "Publishing agent: %v\n", meta)

	// Start publishing
	if err := c.Publish(cmd.Context(), meta, opts.Local); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	presenter.Printf(cmd, "Successfully published!\n")

	return nil
}
