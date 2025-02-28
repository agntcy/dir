// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "pull",
	Short: "Pull compiled agent model from Directory",
	Long: `Usage example:

	# Pull by digest and output in JSON format
	dirctl pull --digest <digest-string> --json

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
		return errors.New("failed to get client from context")
	}

	// Fetch object from store
	reader, err := c.Pull(cmd.Context(), &coretypes.ObjectRef{
		Digest:      opts.AgentDigest,
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Annotations: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to pull data: %w", err)
	}

	// Read the data from the reader.
	agentRaw, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Unmarshal the data into an Agent struct for validation only
	var agent coretypes.Agent
	if err := json.Unmarshal(agentRaw, &agent); err != nil {
		return fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	// If JSON flag is set, marshal the agent to JSON and print it
	if opts.JSON {
		output, err := json.MarshalIndent(&agent, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal agent to JSON: %w", err)
		}

		cmd.Print(string(output))

		return nil
	}

	// Print the raw agent data
	presenter.Print(cmd, string(agentRaw))

	return nil
}
