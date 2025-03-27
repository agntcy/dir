// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package info

import (
	"encoding/json"
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "info",
	Short: "Check info about an object in Directory store",
	Long: `Lookup and get basic metadata about an object
pushed to the Directory store.

Usage example:

	dirctl info <digest>

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
	c, ok := util.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Fetch info from store
	info, err := c.Lookup(cmd.Context(), &coretypes.ObjectRef{
		Digest: digest,
	})
	if err != nil {
		return fmt.Errorf("failed to pull data: %w", err)
	}

	// Marshal metadata for nice preview
	output, err := json.MarshalIndent(&info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent to JSON: %w", err)
	}

	// Print the metadata
	presenter.Print(cmd, string(output))

	return nil
}
