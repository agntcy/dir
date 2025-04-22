// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupword
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
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "push",
	Short: "Push agent data model to Directory server",
	Long: `This command pushes the agent data model to local storage 
layer via Directory API. 
The data is stored into content-addressable object store.

Usage examples:

1. From agent data model file

	dirctl push model.json

2. Data from standard input. Useful for piping

	cat model.json | dirctl push --stdin

3. In combination with other commands such as build and pull:

	dirctl build | dirctl push --stdin

	dirctl pull <digest> | dirctl push --stdin

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var fpath string
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			fpath = args[0]
		}

		// get source
		source, err := getReader(fpath, opts.FromStdin)
		if err != nil {
			return err
		}

		return runCommand(cmd, source)
	},
}

func runCommand(cmd *cobra.Command, source io.ReadCloser) error {
	defer source.Close()

	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Unmarshal the content into an Agent struct.
	agent := &coretypes.Agent{}

	_, err := agent.LoadFromReader(source)
	if err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
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

func getReader(fpath string, fromFile bool) (io.ReadCloser, error) {
	if fpath == "" && !fromFile {
		return nil, errors.New("reqired file path or --stdin flag")
	}

	if fpath != "" {
		file, err := os.Open(fpath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", fpath, err)
		}

		return file, nil
	}

	return os.Stdin, nil
}
