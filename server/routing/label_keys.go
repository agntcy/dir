// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

var (
	errEmptyLabel    = errors.New("cannot derive a DHT key from an empty label")
	errBareNamespace = errors.New("cannot derive a DHT key from a bare namespace")
)

// labelKey maps a label to the DHT key it is advertised under.
//
// Every node derives the same key from the same label string, which is what
// lets a searcher find providers of a label it has never seen announced. Only
// the multihash reaches the DHT (Provide hashes the CID), so the codec is
// cosmetic; what matters is that publisher and searcher hash identically.
func labelKey(label types.Label) (cid.Cid, error) {
	normalized := normalizeLabel(label)
	if normalized == "" {
		return cid.Undef, errEmptyLabel
	}

	if isBareNamespace(normalized) {
		return cid.Undef, fmt.Errorf("%w: %q", errBareNamespace, normalized)
	}

	hash, err := mh.Sum([]byte(normalized), mh.SHA2_256, -1)
	if err != nil {
		return cid.Undef, fmt.Errorf("failed to hash label %q: %w", normalized, err)
	}

	return cid.NewCidV1(cid.Raw, hash), nil
}

// expandLabel returns a label together with its ancestors, closest first:
// "/skills/A/B/C" yields "/skills/A/B/C", "/skills/A/B", "/skills/A".
//
// Ancestors are what make prefix search work: a record tagged "/skills/A/B" is
// only findable under "/skills/A" if its holder also advertises that key.
//
// The bare namespace ("/skills") is deliberately excluded. Every node holding
// any skill at all would provide it, so it selects nothing while attracting
// every provider record in the network onto one set of custodians.
//
// Labels outside the known namespaces are returned as-is; their structure is
// not ours to interpret.
func expandLabel(label types.Label) []types.Label {
	normalized := types.Label(normalizeLabel(label))
	if normalized == "" || isBareNamespace(normalized.String()) {
		return nil
	}

	namespace := normalized.Namespace()
	if namespace == "" {
		return []types.Label{normalized}
	}

	segments := splitNonEmpty(normalized.Value())
	if len(segments) == 0 {
		return nil
	}

	expanded := make([]types.Label, 0, len(segments))

	for i := len(segments); i > 0; i-- {
		expanded = append(expanded, types.Label(namespace+strings.Join(segments[:i], "/")))
	}

	return expanded
}

// expandLabels returns the distinct union of expandLabel over every label,
// preserving first-seen order. Records routinely share ancestors, so the
// deduplicated set is far smaller than the sum of the individual expansions.
func expandLabels(labels []types.Label) []types.Label {
	seen := make(map[types.Label]struct{}, len(labels))
	distinct := make([]types.Label, 0, len(labels))

	for _, label := range labels {
		for _, expanded := range expandLabel(label) {
			if _, ok := seen[expanded]; ok {
				continue
			}

			seen[expanded] = struct{}{}

			distinct = append(distinct, expanded)
		}
	}

	return distinct
}

// isBareNamespace reports whether a normalized label is just a namespace root
// such as "/skills". Such a key matches every record that carries any label of
// that kind, so it discriminates nothing while drawing the entire network's
// provider records onto a single set of custodians.
func isBareNamespace(normalized string) bool {
	for _, labelType := range types.AllLabelTypes() {
		if normalized == "/"+labelType.String() {
			return true
		}
	}

	return false
}

// normalizeLabel trims surrounding whitespace and any trailing slash, so
// "/skills/A" and "/skills/A/" resolve to the same DHT key.
func normalizeLabel(label types.Label) string {
	trimmed := strings.TrimSpace(label.String())
	if trimmed == "/" {
		return ""
	}

	return strings.TrimSuffix(trimmed, "/")
}

// splitNonEmpty splits on "/" and drops empty segments, so a label containing
// a doubled slash does not produce a key with an empty path component.
func splitNonEmpty(value string) []string {
	segments := make([]string, 0, strings.Count(value, "/")+1)

	for segment := range strings.SplitSeq(value, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}

	return segments
}
