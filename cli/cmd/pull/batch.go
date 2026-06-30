// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package pull

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util/records"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

// validatePullInvocation enforces the mutually-exclusive selection/destination
// rules shared by single and batch pulls.
func validatePullInvocation(hasArg bool, outputFile, outputDir string, hasFilters bool) error {
	if outputDir != "" {
		if hasArg {
			return errors.New("positional argument and --output-dir are mutually exclusive")
		}

		if outputFile != "" {
			return errors.New("--output-file and --output-dir are mutually exclusive")
		}

		if !hasFilters {
			return errors.New("at least one search filter is required for batch pull (e.g. --name, --module)")
		}

		return nil
	}

	if !hasArg {
		return errors.New("either a CID/name argument or --output-dir (with search filters) is required")
	}

	return nil
}

// runBatchPull searches for records and writes each as a JSON file into the
// output directory.
func runBatchPull(cmd *cobra.Command, c *client.Client) error {
	queries := search.BuildQueries(&opts.Filters)

	recs, err := records.SearchAndPull(cmd.Context(), c, queries, opts.Limit)
	if err != nil {
		return err
	}

	if len(recs) == 0 {
		presenter.PrintSmartf(cmd, "No records matched the search criteria\n")

		return nil
	}

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil { //nolint:mnd
		return fmt.Errorf("failed to create output directory %s: %w", opts.OutputDir, err)
	}

	toWrite := recs
	if !opts.AllVersions {
		toWrite = records.LatestByName(recs)
	}

	seen := make(map[string]int)
	written := 0

	for i, record := range toWrite {
		base := records.BatchFileName(record, i, seen, opts.AllVersions)
		outPath := filepath.Join(opts.OutputDir, base+".json")

		if err := presenter.WriteMessageToFile(outPath, record.GetData()); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		written++
	}

	presenter.PrintSmartf(cmd, "Pulled %d record(s) to %s\n", written, opts.OutputDir)

	return nil
}
