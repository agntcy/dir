// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"golang.org/x/mod/semver"
)

// SanitizeName replaces characters unsafe for filenames with hyphens.
func SanitizeName(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")

	return r.Replace(name)
}

// canonicalVersion returns the version string normalised to a semver-valid form.
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

// LatestByName deduplicates records by name, keeping only the record with
// the highest semver version for each unique name. Records without a
// parseable version are kept unconditionally.
func LatestByName(records []*corev1.Record) []*corev1.Record {
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

// batchFileName returns a sanitised base name for a record inside a batch
// export.
// When allVersions is true the version is appended to keep every version;
// a numeric suffix is added if the base is still not unique.
// When allVersions is false only the name is used (the caller is expected
// to have already deduplicated to the latest version per name).
func batchFileName(record *corev1.Record, index int, seen map[string]int, allVersions bool) string {
	name := record.GetName()
	if name == "" {
		return fmt.Sprintf("record_%d", index)
	}

	base := SanitizeName(name)

	if allVersions {
		if version := record.GetVersion(); version != "" {
			base += "-" + SanitizeName(version)
		}

		count := seen[base]
		seen[base] = count + 1

		if count > 0 {
			base = fmt.Sprintf("%s-%d", base, count)
		}
	}

	return base
}

// DefaultBatchExport provides per-record file writing for formatters that
// do not implement BatchFormatter.
// When allVersions is false (default), records are deduplicated by name
// and only the highest semver version per name is exported.
// When allVersions is true, every record is exported with the version
// included in the filename.
func DefaultBatchExport(f Formatter, records []*corev1.Record, outputDir string, allVersions bool) (int, error) {
	toExport := records
	if !allVersions {
		toExport = LatestByName(records)
	}

	exported := 0
	seen := make(map[string]int)

	for i, record := range toExport {
		base := batchFileName(record, i, seen, allVersions)

		output, err := f.Format(record)
		if err != nil {
			return exported, fmt.Errorf("failed to format record %q: %w", base, err)
		}

		outPath := filepath.Join(outputDir, base+f.FileExtension())

		if err := os.WriteFile(outPath, output, 0o600); err != nil { //nolint:mnd
			return exported, fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		exported++
	}

	return exported, nil
}
