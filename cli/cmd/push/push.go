// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupword,wrapcheck
package push

import (
	"errors"
	"fmt"
	"io"
	"os"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signcmd "github.com/agntcy/dir/cli/cmd/sign"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "push",
	Short: "Push record to Directory server",
	Long: `This command pushes the record to local storage layer via Directory API. The data is stored into
content-addressable object store.

Usage examples:

1. From record file:

	dirctl push model.json

2. Data from standard input. Useful for piping:

	cat model.json | dirctl push --stdin

3. Push with signature:

	dirctl push model.json --sign

4. Output formats:

	# Get CID as JSON
	dirctl push model.json --output json
	
	# Get raw CID for scripting
	CID=$(dirctl push model.json --output raw)
	
	# Push and pipe to publish
	dirctl push model.json --output raw | xargs dirctl routing publish

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var path string
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			path = args[0]
		}

		// get source
		if path == "" && !opts.FromStdin {
			return errors.New("if no path defined --stdin flag must be set")
		}

		// if path is empty, read from stdin
		if path == "" {
			return runCommand(cmd, cmd.InOrStdin())
		}

		// otherwise, read from file
		source, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("could not open file %s: %w", path, err)
		}
		defer source.Close()

		return runCommand(cmd, source)
	},
}

func runCommand(cmd *cobra.Command, source io.Reader) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Read and close the source
	sourceData, err := io.ReadAll(source)
	if err != nil {
		return fmt.Errorf("failed to read source data: %w", err)
	}

	// Load OASF data into a Record
	record, err := corev1.UnmarshalRecord(sourceData)
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

	// Output in the appropriate format
	return presenter.PrintMessage(cmd, "record", "Pushed record with CID", recordRef.GetCid())
}
