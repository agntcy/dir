// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"encoding/json"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/util"

	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "pull",
	Short: "Pull compiled agent model from Directory",
	Long: `Usage example:

	# Pull by digest
	dirctl pull --digest sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef

	# Pull in combination with other commands
	dirctl pull --digest $(dirctl build | dirctl push)

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	// Get the client from the context.
	c, ok := util.GetClientFromContext(cmd.Context())
	if !ok {
		return fmt.Errorf("failed to get client from context")
	}

	var dig coretypes.Digest
	if err := dig.FromString(opts.AgentDigest); err != nil {
		return fmt.Errorf("failed to parse digest: %w", err)
	}

	// Use the client's Pull method to retrieve the data.
	reader, err := c.Pull(cmd.Context(), &dig)
	if err != nil {
		return fmt.Errorf("failed to pull data: %w", err)
	}

	// Read the data from the reader.
	agentRaw, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Unmarshal the data into an Agent struct for validation only
	{
		var agent coretypes.Agent
		if err := json.Unmarshal(agentRaw, &agent); err != nil {
			return fmt.Errorf("failed to unmarshal agent: %w", err)
		}
	}

	// Print to output
	_, err = fmt.Fprint(cmd.OutOrStdout(), string(agentRaw))
	if err != nil {
		return fmt.Errorf("failed to print built data: %w", err)
	}

	return nil
}
