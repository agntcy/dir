// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/v1/ast"
)

var errEvaluationReadsVerdicts = errors.New("reads verdicts, which an evaluation policy cannot: " +
	"verdicts are what an evaluation produces, and an operation policy is what reads them")

// validateOPAModule checks a parsed module for what can be decided at load
// time. It is the Rego counterpart of validateAst.
//
// An evaluation policy cannot read verdicts at all. An operation policy can,
// and then only by a literal key: input.verdicts["<policy id>"], or
// "<policy id>" in input.verdicts. The key has to name a configured evaluation
// policy. A computed key, or any other use of the map, is refused: an alias
// or an iteration would hide which policies the module depends on. The map is
// always populated for every configured evaluation policy, so a key that names
// none of them is a configuration error rather than an unevaluated record.
func validateOPAModule(mod *ast.Module, l layer, evaluationIDs []string) error {
	if mod == nil || !opaVerdictKeys(mod).reads() {
		return nil
	}

	if l != layerOperation {
		return errEvaluationReadsVerdicts
	}

	return validateOPAVerdictKeys(mod, evaluationIDs)
}

// validateOPAVerdictKeys checks every verdict reference in an operation policy.
func validateOPAVerdictKeys(mod *ast.Module, evaluationIDs []string) error {
	known := make(map[string]struct{}, len(evaluationIDs))
	for _, id := range evaluationIDs {
		known[id] = struct{}{}
	}

	configured := formatEvaluationIDs(evaluationIDs)

	// A bare input.verdicts names no key. "<id>" in input.verdicts is one of
	// those plus a membership call that carries the key, so each such call
	// accounts for one bare reference. What remains was reached some other way.
	bare, err := checkVerdictRefs(mod, known, configured)
	if err != nil {
		return err
	}

	permitted, err := checkVerdictMemberships(mod, known, configured)
	if err != nil {
		return err
	}

	if bare > permitted {
		return errVerdictsHidden
	}

	return nil
}

// checkVerdictRefs checks references that name a key and counts those that do
// not. A non-literal key is refused here; a bare reference is left for the
// membership check to account for.
func checkVerdictRefs(mod *ast.Module, known map[string]struct{}, configured string) (int, error) {
	bare := 0

	var refErr error

	ast.WalkRefs(mod, func(ref ast.Ref) bool {
		key, kind := inputMemberOf(ref, celVarVerdicts)
		if kind == refOther {
			return false
		}

		// input.verdicts on its own names no key. input.verdicts[expr] names
		// one that is only known while the policy runs.
		if kind == refWhole {
			if len(ref) > refMemberPos {
				refErr = errVerdictKeyComputed

				return true
			}

			bare++

			return false
		}

		if _, ok := known[key]; ok {
			return false
		}

		refErr = unknownVerdictKey(key, configured)

		return true
	})

	return bare, refErr
}

// checkVerdictMemberships checks "<key>" in input.verdicts and returns how
// many such tests name a configured policy.
func checkVerdictMemberships(mod *ast.Module, known map[string]struct{}, configured string) (int, error) {
	permitted := 0

	var memberErr error

	ast.WalkExprs(mod, func(e *ast.Expr) bool {
		key, ok := verdictMembership(e)
		if !ok {
			return false
		}

		if key == "" {
			memberErr = errVerdictKeyComputed

			return true
		}

		if _, knownKey := known[key]; !knownKey {
			memberErr = unknownVerdictKey(key, configured)

			return true
		}

		permitted++

		return false
	})

	return permitted, memberErr
}

func unknownVerdictKey(key, configured string) error {
	return fmt.Errorf("verdict key %q does not name a configured evaluation policy (configured: %s)", key, configured)
}

// verdictMembership reads the key of "<key>" in input.verdicts.
//
// The second result is whether the expression is that test. An empty key with
// a true result is a membership test whose key is not a literal string.
func verdictMembership(e *ast.Expr) (string, bool) {
	if e == nil || !e.IsCall() || e.Operator().String() != ast.Member.Name {
		return "", false
	}

	key := e.Operand(0)
	collection := e.Operand(1)

	if key == nil || !isBareInputVerdicts(collection) {
		return "", false
	}

	literal, ok := key.Value.(ast.String)
	if !ok {
		return "", true
	}

	return string(literal), true
}

// isBareInputVerdicts reports whether term is the reference input.verdicts
// and nothing past it.
func isBareInputVerdicts(term *ast.Term) bool {
	if term == nil {
		return false
	}

	ref, ok := term.Value.(ast.Ref)
	if !ok {
		return false
	}

	_, kind := inputMemberOf(ref, celVarVerdicts)

	return kind == refWhole && len(ref) <= refMemberPos
}
