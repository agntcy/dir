// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package export

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/cmd/export/format"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "export <cid-or-name[:version][@digest]>",
	Short: "Export a record from the Directory to a local file or stdout",
	Long: `Export pulls a record from the Directory, transforms it to the requested format,
and writes the result to a file or stdout.

The --format flag selects the output format. The built-in "oasf" format outputs the raw OASF
record JSON.

Usage examples:

  # Export raw OASF JSON to stdout
  dirctl export bafyreib...

  # Export to a file
  dirctl export my-agent:1.0 --output-file=./record.json
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExport(cmd, args[0])
	},
}

func runExport(cmd *cobra.Command, input string) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	formatter, err := format.GetFormatter(opts.Format)
	if err != nil {
		return err
	}

	recordCID, err := reference.ResolveToCID(cmd.Context(), c, input)
	if err != nil {
		return err
	}

	record, err := c.Pull(cmd.Context(), &corev1.RecordRef{
		Cid: recordCID,
	})
	if err != nil {
		return fmt.Errorf("failed to pull record: %w", err)
	}

	output, err := formatter.Format(record)
	if err != nil {
		return fmt.Errorf("failed to format record as %s: %w", opts.Format, err)
	}

	if opts.OutputFile != "" {
		outPath := opts.OutputFile
		if filepath.Ext(outPath) == "" {
			outPath += formatter.FileExtension()
		}

		return writeFile(cmd, outPath, output)
	}

	presenter.Print(cmd, string(output))

	return nil
}

func writeFile(cmd *cobra.Command, path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil { //nolint:mnd
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	presenter.PrintSmartf(cmd, "Exported to: %s\n", path)

	return nil
}
