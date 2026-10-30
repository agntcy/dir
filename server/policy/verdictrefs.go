// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"slices"

	"cel.dev/cel-go/cel"
	celast "cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	opaast "github.com/open-policy-agent/opa/v1/ast"
)

// Which verdicts an operation policy reads is answered from its source, at
// load time, for the same reason the fact groups are: the expressions are
// gone by the time a decision is made.
//
// It is asked for two reasons. It decides when the policy may start
// enforcing, because a policy must not require a verdict from an evaluation
// policy that has not finished covering the catalogue — that is the backfill
// latch, which the SQL gate gets by leaving an uncovered policy out of its
// query, and which a hand-written expression cannot get any other way. And it
// is what a misspelled policy ID is checked against.
//
// The direction of caution is the same as for fact groups but for a different
// reason. Reading a reference as naming more verdicts than it does only holds
// enforcement off for longer. Reading it as naming fewer lets the policy
// enforce against a verdict nothing has produced yet, which hides records
// that were never examined.

// verdictRefs is what one policy's source says about the verdicts it reads.
//
// all is not "every key in keys": it means the policy reaches verdicts in a
// way no key can be read from, so every configured evaluation policy has to
// be assumed. keys is then not worth consulting.
type verdictRefs struct {
	keys []string
	all  bool
}

// reads reports whether the policy touches verdicts at all.
func (r verdictRefs) reads() bool { return r.all || len(r.keys) > 0 }

// celVerdictKeys returns what the compiled expression says about verdicts.
//
// A policy reaches a verdict by indexing verdicts with the policy's ID, which
// is the only way to write one: an ID is provider:name, so the colon keeps it
// from being a field name. A select on verdicts therefore names no key that
// can exist, and is left to the ident count to widen along with every other
// reference whose key cannot be read.
func celVerdictKeys(ast *cel.Ast) verdictRefs {
	var (
		keys  []string
		named int
		seen  int
	)

	celast.PostOrderVisit(ast.NativeRep().Expr(), celast.NewExprVisitor(func(e celast.Expr) {
		switch e.Kind() {
		case celast.CallKind:
			key, ok := indexedVerdictKey(e)
			if !ok {
				return
			}

			keys = append(keys, key)
			named++
		case celast.IdentKind:
			if e.AsIdent() == celVarVerdicts {
				seen++
			}
		case celast.UnspecifiedExprKind, celast.SelectKind, celast.ComprehensionKind,
			celast.ListKind, celast.LiteralKind, celast.MapKind, celast.StructKind:
			// Reached through its operands, which are visited in their own right.
		}
	}))

	// Every verdicts[...] visits the verdicts ident underneath it as well, so
	// a surplus ident is one reached some other way: compared whole, iterated,
	// or indexed by something only known while the policy runs.
	if seen > named {
		return verdictRefs{all: true}
	}

	return verdictRefs{keys: sortedSet(keys)}
}

// indexedVerdictKey reads the key out of verdicts["..."], and reports whether
// the call was one.
//
// An index of verdicts by anything but a string literal names no key and is
// left to the ident count to widen, because the key is only known while the
// policy runs.
func indexedVerdictKey(e celast.Expr) (string, bool) {
	call := e.AsCall()
	if call.FunctionName() != operators.Index || len(call.Args()) != 2 {
		return "", false
	}

	if !isVerdictsIdent(call.Args()[0]) {
		return "", false
	}

	arg := call.Args()[1]
	if arg.Kind() != celast.LiteralKind {
		return "", false
	}

	key, ok := arg.AsLiteral().Value().(string)

	return key, ok
}

func isVerdictsIdent(e celast.Expr) bool {
	return e.Kind() == celast.IdentKind && e.AsIdent() == celVarVerdicts
}

// opaVerdictKeys returns what the module says about verdicts.
//
// Every rule counts, not only the one queried, for the same reason the fact
// groups count every rule: a policy that reads a verdict through a helper
// reads it just the same.
func opaVerdictKeys(mod *opaast.Module) verdictRefs {
	var (
		keys []string
		all  bool
	)

	opaast.WalkRefs(mod, func(ref opaast.Ref) bool {
		key, kind := inputMemberOf(ref, celVarVerdicts)

		switch kind {
		case refMember:
			keys = append(keys, key)
		case refWhole:
			all = true
		case refOther:
		}

		return false
	})

	if all {
		return verdictRefs{all: true}
	}

	return verdictRefs{keys: sortedSet(keys)}
}

// sortedSet returns the values sorted with duplicates removed, or nil when
// there are none.
//
// Unlike the fact groups, nothing is dropped for being unrecognized: a key
// that names no configured policy is a mistake to report, not a name to
// quietly ignore, and reporting it is the load-time check's job.
func sortedSet(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	slices.Sort(values)

	return slices.Compact(values)
}

// celReadsRecord reports whether the expression reads the record's content.
//
// record_cid is a separate name, so a policy that only identifies the record
// does not count as reading it.
func celReadsRecord(ast *cel.Ast) bool {
	reads := false

	celast.PostOrderVisit(ast.NativeRep().Expr(), celast.NewExprVisitor(func(e celast.Expr) {
		if e.Kind() == celast.IdentKind && e.AsIdent() == celVarRecord {
			reads = true
		}
	}))

	return reads
}

// opaReadsRecord reports whether the module reads the record's content.
func opaReadsRecord(mod *opaast.Module) bool {
	reads := false

	opaast.WalkRefs(mod, func(ref opaast.Ref) bool {
		if _, kind := inputMemberOf(ref, celVarRecord); kind != refOther {
			reads = true
		}

		return false
	})

	return reads
}
