// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupword
package push

import (
	"errors"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signcmd "github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/presenter"
	agentUtils "github.com/agntcy/dir/cli/util/agent"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "push",
	Short: "Push agent data model to Directory server",
	Long: `This command pushes the agent data model to local storage layer via Directory API. The data is stored into
content-addressable object store.

Usage examples:

1. From agent data model file:

	dirctl push model.json

2. Data from standard input. Useful for piping:

	cat model.json | dirctl push --stdin

3. Push with signature:

	dirctl push model.json --sign

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var path string
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			path = args[0]
		}

		// get source
		source, err := agentUtils.GetReader(path, opts.FromStdin)
		if err != nil {
			return err //nolint:wrapcheck // Error is already wrapped
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

	// Load OASF data (supports v1, v2, v3) into a Record
	record, err := corev1.LoadOASFFromReader(source)
	if err != nil {
		return fmt.Errorf("failed to load OASF: %w", err)
	}

	var recordRef *corev1.RecordRef

	// Use the client's Push method to send the record
	recordRef, err = c.Push(cmd.Context(), record)
	if err != nil {
		return fmt.Errorf("failed to push data: %w", err)
	}

	if opts.Sign {
		err = signcmd.Sign(cmd.Context(), c, recordRef.GetCid())
		if err != nil {
			return fmt.Errorf("failed to sign record: %w", err)
		}
	}

	presenter.Print(cmd, recordRef.GetCid())

	return nil
}
