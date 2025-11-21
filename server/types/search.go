// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import searchv1 "github.com/agntcy/dir/api/search/v1"

// QueryExpression represents a boolean expression tree for search queries.
// This is an internal domain model - not exposed via API.
type QueryExpression struct {
	// Only one of these should be set
	Query *searchv1.RecordQuery
	And   *AndExpression
	Or    *OrExpression
	Not   *NotExpression
}

// AndExpression represents a logical AND of multiple expressions.
// A record matches if ALL sub-expressions evaluate to true.
type AndExpression struct {
	Expressions []*QueryExpression
}

// OrExpression represents a logical OR of multiple expressions.
// A record matches if ANY sub-expression evaluates to true.
type OrExpression struct {
	Expressions []*QueryExpression
}

// NotExpression represents a logical NOT of an expression.
// A record matches if the sub-expression evaluates to false.
type NotExpression struct {
	Expression *QueryExpression
}
