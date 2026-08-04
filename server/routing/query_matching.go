// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"strings"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var queryLogger = logging.Logger("routing/query")

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
			// Exact match: /modules/runtime/model matches "runtime/model"
			if labelStr == targetModule {
				return true
			}
			// Prefix match: /modules/runtime/model/security matches "runtime/model"
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
