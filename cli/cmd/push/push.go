// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	registrytypes "github.com/agntcy/dir/api/registry/v1alpha1"
	"github.com/agntcy/dir/cli/util"

	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "push",
	Short: "Push compiled agent model to registry server",
	Long: `Usage example:

	# From file
	dirctl push --from-file compiled.json

	# From stdin
	dirctl build <args> | dirctl push

	# Forward (no-op)
	dirctl pull --id agent-id | dirctl push

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	// Get the registry client from the context.
	c, ok := util.GetRegistryClientFromContext(cmd.Context())
	if !ok {
		return fmt.Errorf("failed to get registry client from context")
	}

	// Create a reader from the file or stdin.
	reader, err := getReader()
	if err != nil {
		return fmt.Errorf("could not create reader: %w", err)
	}

	// Unmarshal the content into an Agent struct.
	agent, err := unmarshalAgent(reader)
	if err != nil {
		return fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	// Marshal the Agent struct back to bytes.
	data, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	// Define the metadata for the object.
	meta := &registrytypes.ObjectMeta{
		Type:        registrytypes.ObjectType_OBJECT_TYPE_AGENT,
		Name:        agent.Name,
		Annotations: agent.Annotations,
	}

	// Use the client's Push method to send the data.
	digest, err := c.Push(cmd.Context(), meta, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to push data: %w", err)
	}

	fmt.Printf("Pushed data with digest: %v\n", digest.ToString())
	return nil
}

func getReader() (io.Reader, error) {
	if opts.FromFile != "" {
		return os.Open(opts.FromFile)
	}

	return os.Stdin, nil
}

func unmarshalAgent(reader io.Reader) (*coretypes.Agent, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var agent coretypes.Agent
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	return &agent, nil
}
