// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package records holds helpers shared by the commands that retrieve records
// from the Directory in bulk (pull, export) and turn them into files: search →
// CIDs → records, plus filename derivation and latest-per-name deduplication.
package records

import (
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client"
	"golang.org/x/mod/semver"
)

// CollectCIDs runs a search and returns the matching record CIDs.
func CollectCIDs(ctx context.Context, c *client.Client, queries []*searchv1.RecordQuery, limit uint32) ([]string, error) {
	result, err := c.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{
		Limit:   &limit,
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
		case <-ctx.Done():
			return nil, fmt.Errorf("search cancelled: %w", ctx.Err())
		}
	}
}

// SearchAndPull runs a search and pulls every matching record.
func SearchAndPull(ctx context.Context, c *client.Client, queries []*searchv1.RecordQuery, limit uint32) ([]*corev1.Record, error) {
	cids, err := CollectCIDs(ctx, c, queries, limit)
	if err != nil {
		return nil, err
	}

	records := make([]*corev1.Record, 0, len(cids))

	for _, cid := range cids {
		record, err := c.Pull(ctx, &corev1.RecordRef{Cid: cid})
		if err != nil {
			return nil, fmt.Errorf("failed to pull record %s: %w", cid, err)
		}

		records = append(records, record)
	}

	return records, nil
}

// SanitizeName replaces characters unsafe for filenames with hyphens.
func SanitizeName(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")

	return r.Replace(name)
}

// LatestByName deduplicates records by name, keeping the highest semver version
// for each unique name while preserving first-seen order.
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

// BatchFileName derives a stable, filesystem-safe base name for a record. When
// allVersions is set the version is appended to the base. Filename collisions
// are always disambiguated via the seen map (which the caller threads across
// records), so distinct names that sanitize to the same base — or repeated
// name+version pairs — never overwrite one another.
func BatchFileName(record *corev1.Record, index int, seen map[string]int, allVersions bool) string {
	name := record.GetName()
	if name == "" {
		return fmt.Sprintf("record_%d", index)
	}

	base := SanitizeName(name)

	if allVersions {
		if version := record.GetVersion(); version != "" {
			base += "-" + SanitizeName(version)
		}
	}

	count := seen[base]
	seen[base] = count + 1

	if count > 0 {
		base = fmt.Sprintf("%s-%d", base, count)
	}

	return base
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
