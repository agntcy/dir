// Copyright AGNTCY Contributors (https://github.com/agntcy)
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

	# Pull by digest and output it
	dirctl pull <digest>

	# Pull in combination with other commands
	dirctl pull $(dirctl build | dirctl push --stdin)

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

	// Fetch object from store
	reader, err := c.Pull(cmd.Context(), &coretypes.ObjectRef{
		Digest:      digest,
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

	// If raw format flag is set, print and exit
	if opts.FormatRaw {
		presenter.Print(cmd, string(agentRaw))

		return nil
	}

	// Pretty-print the agent
	output, err := json.MarshalIndent(&agent, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent to JSON: %w", err)
	}

	presenter.Print(cmd, string(output))

	return nil
}
