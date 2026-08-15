// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"

	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
)

// Result is the extractor output: skills, domains, modules, and keywords found
// in the input text. It is the SDK type so downstream consumers (e.g. the
// nlsearch decomposer) work unchanged regardless of which backend produced it.
type Result = sdk.Result

// ExtractOptions carries the backend-agnostic, per-query knobs. It deliberately
// omits construction-time tuning (embedding weights, default thresholds), which
// only a local extractor can honor and are supplied when it is built.
type ExtractOptions struct {
	// Versions pins the OASF schema versions to consider. Empty means all.
	Versions []string

	// Tiers is the number of score tiers (score groups) to return per kind.
	// 0 uses DefaultTiers; higher values widen recall (each tier is the next
	// closest group of matches).
	Tiers int
}

// DefaultTiers is the number of score tiers the extractor returns per kind when
// a caller does not set ExtractOptions.Tiers. It is 2 (the two closest groups)
// rather than the SDK's own default of 1, so search and enrichment favour recall
// across the whole platform. Callers can still narrow (Tiers: 1) or widen it.
const DefaultTiers = 2

// Extractor turns free-form text into OASF skills, domains, modules, and
// keywords. It is satisfied by both the in-process local backend and the remote
// gRPC OASF-SDK backend, so callers depend only on this interface.
type Extractor interface {
	Extract(ctx context.Context, text string, opts ExtractOptions) (Result, error)
	// Close releases resources held by the extractor, such as the remote
	// backend's gRPC connection. Callers that resolve an extractor own its
	// lifecycle and should Close it when done. The local backend holds no
	// closable resources and returns nil.
	Close() error
}
