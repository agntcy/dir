// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ranking

import "sort"

// SortDescending orders a slice of Scored items by composite Score
// DESC, with CID ASC as a deterministic tiebreaker. Sorting is in-place
// and stable for items with equal scores AND equal CIDs (a pathological
// case in practice).
//
// Without a tiebreaker, ties would resolve by insertion order, which
// happens to be SQL row order today but is not contractual. The CID
// tiebreaker makes results reproducible across requests and across
// machines.
func SortDescending(items []Scored) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Result.Score != items[j].Result.Score {
			return items[i].Result.Score > items[j].Result.Score
		}

		return items[i].CID < items[j].CID
	})
}

// Paginate returns items[offset:offset+limit], clamping offset and
// limit safely:
//   - offset < 0 is treated as 0
//   - offset >= len(items) returns an empty slice
//   - limit <= 0 returns everything from offset onward
//   - limit > remaining returns everything from offset onward
//
// The returned slice shares backing storage with items; callers that
// mutate it should copy first.
func Paginate(items []Scored, offset, limit int) []Scored {
	if offset < 0 {
		offset = 0
	}

	if offset >= len(items) {
		return nil
	}

	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return items[offset:end]
}
