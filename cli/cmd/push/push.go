// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package push

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "push",
	Short: "Push compiled agent model to Directory server",
	Long: `Usage example:

	# From file
	dirctl push --from-file compiled.json

	# From stdin
	dirctl build <args> | dirctl push

	# From pull
	dirctl pull --digest <digest-string> --json | dirctl push

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

	// Use the client's Push method to send the data.
	ref, err := c.Push(cmd.Context(), &coretypes.ObjectRef{
		Digest:      digest.FromBytes(data).String(),
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Size:        uint64(len(data)),
		Annotations: agent.GetAnnotations(),
	}, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to push data: %w", err)
	}

	// Print digest to output
	presenter.Print(cmd, ref.GetDigest())

	return nil
}

func getReader() (io.Reader, error) {
	if opts.FromFile != "" {
		file, err := os.Open(opts.FromFile)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", opts.FromFile, err)
		}

		return file, nil
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
