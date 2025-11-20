// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/server/types"
)

// ConvertQueriesToExpression converts a flat list of queries into an expression tree.
// The conversion follows these rules:
// 1. All queries are AND'd together (every condition must match)
//
// Example: [name="web*", version="1.0", skill="python*"]
// Becomes: AND(name="web*", version="1.0", skill="python*")
//
// This preserves the existing search behavior where all queries must match.
func ConvertQueriesToExpression(queries []*searchv1.RecordQuery) *types.QueryExpression {
	if len(queries) == 0 {
		return nil
	}

	// Single query - no need for AND expression
	if len(queries) == 1 {
		return &types.QueryExpression{
			Query: queries[0],
		}
	}

	// Multiple queries - AND them all together
	var expressions []*types.QueryExpression
	for _, q := range queries {
		expressions = append(expressions, &types.QueryExpression{
			Query: q,
		})
	}

	return &types.QueryExpression{
		And: &types.AndExpression{
			Expressions: expressions,
		},
	}
}
