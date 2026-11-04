// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package predicate holds what a policy's expression compiles to when it has
// to be enforced over many records at once.
//
// A read by CID can evaluate a policy directly: the record is identified, its
// verdicts can be fetched, and the expression runs as written. A search
// cannot. The records it would return are what the policy decides, so the
// policy has to become part of the query rather than a filter over its
// result, and that means it has to become SQL.
//
// Only boolean combinations of verdict terms survive that translation, so
// that is all a Node can express. Everything else a policy might say is
// refused when the policy is loaded, where the author can be told why.
package predicate

// Node is a boolean combination of verdict terms.
//
// The set of kinds is closed: isNode is unexported, so only this package
// defines one. The kinds themselves are exported because the translation to
// SQL belongs with the schema it writes against, not here.
type Node interface {
	isNode()
}

// Verdict requires an evaluated, compliant verdict for the record under
// PolicyID.
//
// The version is deliberately absent. A policy's expression names the policy
// it reads a verdict of, not the version it reads; which version is in force
// changes as evaluators register, long after this tree was built. Whatever is
// in force when the query runs is what the term means.
type Verdict struct {
	PolicyID string
}

// All holds when every part holds.
type All struct {
	Parts []Node
}

// Any holds when at least one part holds.
type Any struct {
	Parts []Node
}

func (Verdict) isNode() {}
func (All) isNode()     {}
func (Any) isNode()     {}
