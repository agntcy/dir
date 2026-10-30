// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"slices"

	"cel.dev/cel-go/cel"
	celast "cel.dev/cel-go/common/ast"
	"github.com/agntcy/dir/server/types"
	opaast "github.com/open-policy-agent/opa/v1/ast"
)

// Which fact groups a policy reads is answered from its source, at load time,
// because that is the only place the question has an answer: by the time a
// verdict exists, the expressions that produced it are gone.
//
// Both engines are read the same way, and both can be read truthfully. The
// care here is all in one direction. Reporting a group a policy does not read
// costs a re-evaluation that reaches the same answer. Missing one a policy
// does read means its verdict is never invalidated and is served as current
// forever, with nothing to notice. So anything that cannot be resolved to a
// definite set of groups is taken to be all of them, and every such case is
// named below rather than falling through.

// celFactGroups returns the groups the compiled expression reads.
//
// A policy reaches a fact by selecting a field of meta, so the groups are the
// field names of selects whose operand is meta itself. meta reached any other
// way — compared whole, passed to a function, indexed — is a reference the
// field name cannot be read from, and means every group.
func celFactGroups(ast *cel.Ast) []string {
	var (
		groups  []string
		selects int
		idents  int
	)

	celast.PostOrderVisit(ast.NativeRep().Expr(), celast.NewExprVisitor(func(e celast.Expr) {
		switch e.Kind() {
		case celast.SelectKind:
			sel := e.AsSelect()
			if isMetaIdent(sel.Operand()) {
				groups = append(groups, sel.FieldName())
				selects++
			}
		case celast.IdentKind:
			if e.AsIdent() == celVarMeta {
				idents++
			}
		case celast.UnspecifiedExprKind, celast.CallKind, celast.ComprehensionKind,
			celast.ListKind, celast.LiteralKind, celast.MapKind, celast.StructKind:
			// Reached through its operands, which are visited in their own right.
		}
	}))

	// Every meta.<group> select visits the meta ident underneath it as well,
	// so a surplus ident is one reached some other way.
	if idents > selects {
		return types.FactGroups()
	}

	return normalizeGroups(groups)
}

func isMetaIdent(e celast.Expr) bool {
	return e.Kind() == celast.IdentKind && e.AsIdent() == celVarMeta
}

// opaFactGroups returns the groups the module reads.
//
// Every rule in the module counts, not only the one queried: a policy that
// reads a fact through a helper reads it just the same. A reference that stops
// at input.meta, or that selects the group dynamically, names no group and so
// means every group.
func opaFactGroups(mod *opaast.Module) []string {
	var (
		groups []string
		all    bool
	)

	opaast.WalkRefs(mod, func(ref opaast.Ref) bool {
		group, kind := inputMemberOf(ref, celVarMeta)

		switch kind {
		case refMember:
			groups = append(groups, group)
		case refWhole:
			all = true
		case refOther:
		}

		return false
	})

	if all {
		return types.FactGroups()
	}

	return normalizeGroups(groups)
}

type refKind int

const (
	// refOther is a reference to something else entirely.
	refOther refKind = iota

	// refWhole reaches the field but names no member of it.
	refWhole

	// refMember names one member of the field.
	refMember
)

// A reference a policy writes as input.meta.scan, or input.verdicts["cel:x"],
// is the terms input, the field, and the member, in that order.
const (
	refFieldPos  = 1
	refMemberPos = 2
)

// inputMemberOf classifies one reference against one field of input:
// input.<field>.<member> names a member, input.<field> on its own or with a
// member this cannot read names the whole field, and anything else is not
// about the field at all.
//
// Both callers read a field whose members are a known set, and both are safe
// only when a reference they cannot resolve widens the answer rather than
// narrowing it, which is what refWhole is for.
func inputMemberOf(ref opaast.Ref, field string) (string, refKind) {
	if len(ref) <= refFieldPos || !ref[0].Equal(opaast.InputRootDocument) {
		return "", refOther
	}

	if name, ok := ref[refFieldPos].Value.(opaast.String); !ok || string(name) != field {
		return "", refOther
	}

	if len(ref) <= refMemberPos {
		return "", refWhole
	}

	member, ok := ref[refMemberPos].Value.(opaast.String)
	if !ok {
		// A variable or expression in place of the member, as in
		// input.meta[g], which is only known while the policy runs.
		return "", refWhole
	}

	return string(member), refMember
}

// normalizeGroups returns the groups as a sorted set, dropping any name that
// is not a group this node knows.
//
// An unknown name is dropped rather than widening the result: the environments
// reject a misspelled field at load time, so a name reaching here that is not
// a group is a field of meta that carries no facts rather than a fact this
// cannot see.
func normalizeGroups(groups []string) []string {
	if len(groups) == 0 {
		return nil
	}

	known := types.FactGroups()
	out := make([]string, 0, len(groups))

	for _, g := range groups {
		if slices.Contains(known, g) && !slices.Contains(out, g) {
			out = append(out, g)
		}
	}

	if len(out) == 0 {
		return nil
	}

	slices.Sort(out)

	return out
}
