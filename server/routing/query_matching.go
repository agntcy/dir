// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"strings"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var queryLogger = logging.Logger("routing/query")

// LabelRetriever function type for injecting different label retrieval strategies.
// This allows us to use the same query matching logic for both local and remote scenarios
// while keeping the label retrieval implementation separate.
type LabelRetriever func(ctx context.Context, cid string) []types.Label

// MatchesAllQueries checks if a record matches ALL provided queries using injected label retrieval.
// This implements AND logic - all queries must match for the record to be considered a match.
//
// Parameters:
//   - ctx: Context for the operation
//   - cid: The CID of the record to check
//   - queries: List of queries that must ALL match (AND relationship)
//   - labelRetriever: Function to retrieve labels for the given CID
//
// Returns true if all queries match, false otherwise.
func MatchesAllQueries(
	ctx context.Context,
	cid string,
	queries []*routingv1.RecordQuery,
	labelRetriever LabelRetriever,
) bool {
	if len(queries) == 0 {
		return true // No filters = match everything
	}

	// Use the injected label retrieval strategy
	labels := labelRetriever(ctx, cid)

	// ALL queries must match (AND relationship)
	for _, query := range queries {
		if !QueryMatchesLabels(query, labels) {
			return false
		}
	}

	return true
}

// QueryMatchesLabels checks if a single query matches against a list of labels.
// This function contains the unified logic for all query types, resolving the
// differences between local and remote implementations.
//
//nolint:gocognit,cyclop // Complex but necessary logic for handling all query types with exact and prefix matching
func QueryMatchesLabels(query *routingv1.RecordQuery, labelList []types.Label) bool {
	if query == nil {
		return false
	}

	switch query.GetType() {
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL:
		// Check if any skill label matches the query
		targetSkill := types.LabelTypeSkill.Prefix() + query.GetValue()

		for _, label := range labelList {
			// Type-safe filtering: only check skill labels
			if label.Type() != types.LabelTypeSkill {
				continue
			}

			labelStr := label.String()
			// Exact match: /skills/category1/class1 matches "category1/class1"
			if labelStr == targetSkill {
				return true
			}
			// Prefix match: /skills/category2/class2 matches "category2"
			if strings.HasPrefix(labelStr, targetSkill+"/") {
				return true
			}
		}

		return false

	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR:
		// Unified locator handling - use proper namespace prefix (fixing remote implementation)
		targetLocator := types.LabelTypeLocator.Prefix() + query.GetValue()

		for _, label := range labelList {
			// Type-safe filtering: only check locator labels
			if label.Type() != types.LabelTypeLocator {
				continue
			}

			// Exact match: /locators/docker-image matches "docker-image"
			if label.String() == targetLocator {
				return true
			}
		}

		return false

	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN:
		// Check if any domain label matches the query
		targetDomain := types.LabelTypeDomain.Prefix() + query.GetValue()

		for _, label := range labelList {
			// Type-safe filtering: only check domain labels
			if label.Type() != types.LabelTypeDomain {
				continue
			}

			labelStr := label.String()
			// Exact match: /domains/research matches "research"
			if labelStr == targetDomain {
				return true
			}
			// Prefix match: /domains/research/subfield matches "research"
			if strings.HasPrefix(labelStr, targetDomain+"/") {
				return true
			}
		}

		return false

	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE:
		// Check if any module label matches the query
		targetModule := types.LabelTypeModule.Prefix() + query.GetValue()

		for _, label := range labelList {
			// Type-safe filtering: only check module labels
			if label.Type() != types.LabelTypeModule {
				continue
			}

			labelStr := label.String()
			// Exact match: /modules/runtime/language matches "runtime/language"
			if labelStr == targetModule {
				return true
			}
			// Prefix match: /modules/runtime/language/python matches "runtime/language"
			if strings.HasPrefix(labelStr, targetModule+"/") {
				return true
			}
		}

		return false

	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED:
		// Unspecified queries match everything
		return true

	default:
		queryLogger.Warn("Unknown query type", "type", query.GetType())

		return false
	}
}

// GetMatchingQueries returns the queries that match against a specific label key.
// This is used primarily for calculating match scores in Search operations.
func GetMatchingQueries(labelKey string, queries []*routingv1.RecordQuery) []*routingv1.RecordQuery {
	var matchingQueries []*routingv1.RecordQuery

	// Extract label from the enhanced key
	label, _, _, err := ParseEnhancedLabelKey(labelKey)
	if err != nil {
		queryLogger.Warn("Failed to parse enhanced label key for query matching", "key", labelKey, "error", err)

		return matchingQueries
	}

	// Check which queries this label satisfies
	for _, query := range queries {
		if QueryMatchesLabels(query, []types.Label{label}) {
			matchingQueries = append(matchingQueries, query)
		}
	}

	return matchingQueries
}
