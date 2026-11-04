// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"errors"
	"fmt"

	celast "cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"github.com/agntcy/dir/server/policy/predicate"
)

// A policy at search is the one kind that is never evaluated. Its decision
// has to be known before rows are selected, so it is translated into the
// query instead, and what cannot be translated is refused while the operator
// is still reading the error.
//
// The subset is deliberately the smallest one that says anything: boolean
// combinations of one verdict field. Every construct outside it is refused
// not by a check that looks for it, but by there being no case that handles
// it, so widening the subset is always a deliberate addition.

// verdictFieldCompliantAndCurrent is the only field of a verdict a search
// policy may read.
//
// It is one field because it bundles three conditions that have to travel
// together: evaluated, compliant, and at the version in force. Reading them
// apart would let a policy ask about a verdict under a superseded version,
// which is not knowledge about the policy in force.
const verdictFieldCompliantAndCurrent = "compliant_and_current"

// celSearchPredicate compiles one expression into the predicate a search
// enforces.
func celSearchPredicate(e celast.Expr) (predicate.Node, error) {
	switch e.Kind() {
	case celast.SelectKind:
		return celVerdictTerm(e)
	case celast.CallKind:
		return celSearchCall(e)
	case celast.UnspecifiedExprKind, celast.IdentKind, celast.LiteralKind,
		celast.ListKind, celast.MapKind, celast.StructKind, celast.ComprehensionKind:
		return nil, fmt.Errorf("%w: expected a verdict, or verdicts combined with && or ||", errNotCompilable)
	default:
		return nil, fmt.Errorf("%w: expected a verdict, or verdicts combined with && or ||", errNotCompilable)
	}
}

// errNotCompilable is what every refusal wraps, so that the registry can say
// where the policy was refused and this can say why.
var errNotCompilable = errors.New("not expressible as a query")

func celSearchCall(e celast.Expr) (predicate.Node, error) {
	call := e.AsCall()

	switch call.FunctionName() {
	case operators.LogicalAnd:
		parts, err := celSearchParts(call.Args())
		if err != nil {
			return nil, err
		}

		return predicate.All{Parts: parts}, nil
	case operators.LogicalOr:
		parts, err := celSearchParts(call.Args())
		if err != nil {
			return nil, err
		}

		return predicate.Any{Parts: parts}, nil
	default:
		return nil, fmt.Errorf("%w: %q is not available at search, only && and ||",
			errNotCompilable, call.FunctionName())
	}
}

func celSearchParts(args []celast.Expr) ([]predicate.Node, error) {
	parts := make([]predicate.Node, 0, len(args))

	for _, arg := range args {
		part, err := celSearchPredicate(arg)
		if err != nil {
			return nil, err
		}

		parts = append(parts, part)
	}

	return parts, nil
}

// celVerdictTerm reads verdicts["id"].compliant_and_current.
func celVerdictTerm(e celast.Expr) (predicate.Node, error) {
	sel := e.AsSelect()

	// has(verdicts["id"].compliant_and_current) asks whether the field is
	// set, which is not what the query can ask.
	if sel.IsTestOnly() {
		return nil, fmt.Errorf("%w: has() is not available at search", errNotCompilable)
	}

	if sel.FieldName() != verdictFieldCompliantAndCurrent {
		return nil, fmt.Errorf("%w: a search policy can read only %s of a verdict, not %s",
			errNotCompilable, verdictFieldCompliantAndCurrent, sel.FieldName())
	}

	key, ok := indexedVerdictKey(sel.Operand())
	if !ok {
		return nil, fmt.Errorf("%w: a verdict has to be read as %s[\"id\"].%s, with the id written out",
			errNotCompilable, celVarVerdicts, verdictFieldCompliantAndCurrent)
	}

	return predicate.Verdict{PolicyID: key}, nil
}

// celSearchPredicates compiles a policy's expressions into one predicate.
//
// A policy's expressions all have to hold, so together they are a
// conjunction, the same reading every other layer gives them.
func celSearchPredicates(nodes []predicate.Node) predicate.Node {
	if len(nodes) == 1 {
		return nodes[0]
	}

	return predicate.All{Parts: nodes}
}
