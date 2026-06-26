// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package pull

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
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

	if outputFile != "" && !hasArg {
		return errors.New("--output-file requires a CID/name argument")
	}

	return nil
}

// runBatchPull searches for records and writes each as a JSON file into the
// output directory.
func runBatchPull(cmd *cobra.Command, c *client.Client) error {
	queries := search.BuildQueries(&opts.Filters)

	records, err := searchAndPull(cmd, c, queries)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		presenter.PrintSmartf(cmd, "No records matched the search criteria\n")

		return nil
	}

	toWrite := records
	if !opts.AllVersions {
		toWrite = latestByName(records)
	}

	seen := make(map[string]int)
	written := 0

	for i, record := range toWrite {
		base := batchFileName(record, i, seen, opts.AllVersions)
		outPath := filepath.Join(opts.OutputDir, base+".json")

		if err := presenter.WriteMessageToFile(outPath, record.GetData()); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		written++
	}

	presenter.PrintSmartf(cmd, "Pulled %d record(s) to %s\n", written, opts.OutputDir)

	return nil
}

func searchAndPull(cmd *cobra.Command, c *client.Client, queries []*searchv1.RecordQuery) ([]*corev1.Record, error) {
	cids, err := collectCIDs(cmd, c, queries)
	if err != nil {
		return nil, err
	}

	records := make([]*corev1.Record, 0, len(cids))

	for _, cid := range cids {
		record, err := c.Pull(cmd.Context(), &corev1.RecordRef{Cid: cid})
		if err != nil {
			return nil, fmt.Errorf("failed to pull record %s: %w", cid, err)
		}

		records = append(records, record)
	}

	return records, nil
}

func collectCIDs(cmd *cobra.Command, c *client.Client, queries []*searchv1.RecordQuery) ([]string, error) {
	result, err := c.SearchCIDs(cmd.Context(), &searchv1.SearchCIDsRequest{
		Limit:   &opts.Limit,
		Queries: queries,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var cids []string

	for {
		select {
		case resp := <-result.ResCh():
			if cid := resp.GetRecordCid(); cid != "" {
				cids = append(cids, cid)
			}
		case err := <-result.ErrCh():
			return nil, fmt.Errorf("error during search: %w", err)
		case <-result.DoneCh():
			return cids, nil
		case <-cmd.Context().Done():
			return nil, cmd.Context().Err()
		}
	}
}

// sanitizeName replaces characters unsafe for filenames with hyphens.
func sanitizeName(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")

	return r.Replace(name)
}

func canonicalVersion(raw string) string {
	v := raw
	if v != "" && v[0] != 'v' {
		v = "v" + v
	}

	if semver.IsValid(v) {
		return v
	}

	return ""
}

// latestByName deduplicates records by name, keeping the highest semver version.
func latestByName(records []*corev1.Record) []*corev1.Record {
	type entry struct {
		record  *corev1.Record
		version string
	}

	best := map[string]*entry{}

	var order []string

	for _, r := range records {
		name := r.GetName()
		ver := canonicalVersion(r.GetVersion())

		existing, seen := best[name]
		if !seen {
			order = append(order, name)
			best[name] = &entry{record: r, version: ver}

			continue
		}

		if ver == "" {
			continue
		}

		if existing.version == "" || semver.Compare(ver, existing.version) > 0 {
			best[name] = &entry{record: r, version: ver}
		}
	}

	result := make([]*corev1.Record, 0, len(order))
	for _, name := range order {
		result = append(result, best[name].record)
	}

	return result
}

func batchFileName(record *corev1.Record, index int, seen map[string]int, allVersions bool) string {
	name := record.GetName()
	if name == "" {
		return fmt.Sprintf("record_%d", index)
	}

	base := sanitizeName(name)

	if allVersions {
		if version := record.GetVersion(); version != "" {
			base += "-" + sanitizeName(version)
		}

		count := seen[base]
		seen[base] = count + 1

		if count > 0 {
			base = fmt.Sprintf("%s-%d", base, count)
		}
	}

	return base
}
