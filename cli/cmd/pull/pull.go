// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"encoding/json"
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "pull",
	Short: "Pull agent model from Directory server",
	Long: `This command pulls the agent data model from Directory API. 
The data can be validated against its hash, as the returned object
is content-addressable.

Usage examples:

1. Pull by digest and output

	dirctl pull <digest>

2. In combination with other commands such as build and push:

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
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
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

	// Load into an Agent struct for validation only
	agent := &coretypes.Agent{}

	agentRaw, err := agent.LoadFromReader(reader)
	if err != nil {
		return fmt.Errorf("failed to load agent from reader: %w", err)
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
