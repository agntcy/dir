// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"errors"
	"fmt"
	"strings"

	"cel.dev/cel-go/cel"
	celast "cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"cel.dev/cel-go/common/types"
)

var (
	errVerdictKeyComputed = errors.New("verdict key is not a literal string, so it cannot be checked against the configured evaluation policies")
	errVerdictsHidden     = errors.New("verdicts is used without a literal key; iterating or aliasing it hides which evaluation policies this policy depends on")
)

// validateAst checks a compiled expression for what can be decided at load
// time. Further checks belong here.
//
// Today it checks verdict keys. The only accepted reads are a literal index,
// verdicts["<policy id>"], and a literal membership test, "<policy id>" in
// verdicts. A computed key, or any other use of the map, is refused: an alias
// or a comprehension would hide which policies the expression depends on. The
// map is always populated for every configured evaluation policy, so a key
// that names none of them is a configuration error rather than an unevaluated
// record.
func validateAst(ast *cel.Ast, evaluationIDs []string) error {
	if ast == nil || ast.NativeRep() == nil {
		return nil
	}

	known := make(map[string]struct{}, len(evaluationIDs))
	for _, id := range evaluationIDs {
		known[id] = struct{}{}
	}

	w := &verdictWalk{
		ast:        ast.NativeRep(),
		known:      known,
		configured: formatEvaluationIDs(evaluationIDs),
		permitted:  map[int64]struct{}{},
	}
	celast.PreOrderVisit(w.ast.Expr(), celast.NewExprVisitor(w.visit))

	return w.err
}

func formatEvaluationIDs(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}

	return strings.Join(ids, ", ")
}

// verdictWalk records literal verdict keys and the first use it cannot check.
//
// Preorder matters: a call is visited before its arguments, so an index marks
// its receiver as a permitted use before that identifier is visited on its own.
type verdictWalk struct {
	ast        *celast.AST
	known      map[string]struct{}
	configured string
	permitted  map[int64]struct{}
	err        error
}

func (w *verdictWalk) visit(e celast.Expr) {
	if w.err != nil || e == nil {
		return
	}

	switch e.Kind() {
	case celast.IdentKind:
		w.visitIdent(e)
	case celast.CallKind:
		w.visitCall(e.AsCall())
	case celast.UnspecifiedExprKind, celast.ComprehensionKind, celast.ListKind,
		celast.LiteralKind, celast.MapKind, celast.SelectKind, celast.StructKind:
		// A verdict read is an identifier or a call. The other kinds are
		// reached through those when they belong to one.
	}
}

func (w *verdictWalk) visitIdent(e celast.Expr) {
	if !w.isVerdicts(e) {
		return
	}

	if _, ok := w.permitted[e.ID()]; ok {
		return
	}

	w.err = errVerdictsHidden
}

func (w *verdictWalk) visitCall(call celast.CallExpr) {
	recv, key, ok := verdictAccess(call)
	if !ok || !w.isVerdicts(recv) {
		return
	}

	w.permitted[recv.ID()] = struct{}{}

	literal, ok := literalString(key)
	if !ok {
		w.err = errVerdictKeyComputed

		return
	}

	if _, known := w.known[literal]; known {
		return
	}

	w.err = fmt.Errorf("verdict key %q does not name a configured evaluation policy (configured: %s)", literal, w.configured)
}

func (w *verdictWalk) isVerdicts(e celast.Expr) bool {
	if e == nil || e.Kind() != celast.IdentKind || e.AsIdent() != celVarVerdicts {
		return false
	}

	// A comprehension variable can be named verdicts. The map is the one
	// whose type is the verdict map; the loop variable is an element.
	return w.ast.GetType(e.ID()).Kind() == types.MapKind
}

// nonMemberArgCount is the argument count of a non-member index, map[key],
// and of a membership test, key in map. A member index carries its receiver
// separately and has one argument.
const nonMemberArgCount = 2

// verdictAccess returns the map and the key of a verdict read.
// Index is written map[key]. Membership is written key in map.
func verdictAccess(call celast.CallExpr) (celast.Expr, celast.Expr, bool) {
	switch call.FunctionName() {
	case operators.Index, operators.OptIndex:
		return indexArgs(call)
	case operators.In, operators.OldIn:
		return inArgs(call)
	default:
		return nil, nil, false
	}
}

func indexArgs(call celast.CallExpr) (celast.Expr, celast.Expr, bool) {
	if call.IsMemberFunction() {
		args := call.Args()
		if call.Target() == nil || len(args) != 1 {
			return nil, nil, false
		}

		return call.Target(), args[0], true
	}

	args := call.Args()
	if len(args) != nonMemberArgCount {
		return nil, nil, false
	}

	return args[0], args[1], true
}

func inArgs(call celast.CallExpr) (celast.Expr, celast.Expr, bool) {
	args := call.Args()
	if call.IsMemberFunction() || len(args) != nonMemberArgCount {
		return nil, nil, false
	}

	return args[1], args[0], true
}

func literalString(e celast.Expr) (string, bool) {
	if e == nil || e.Kind() != celast.LiteralKind || e.AsLiteral() == nil {
		return "", false
	}

	s, ok := e.AsLiteral().Value().(string)

	return s, ok
}
