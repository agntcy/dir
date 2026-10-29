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
		group, kind := metaGroupOf(ref)

		switch kind {
		case refGroup:
			groups = append(groups, group)
		case refWholeMeta:
			all = true
		case refNotMeta:
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
	refNotMeta refKind = iota
	refWholeMeta
	refGroup
)

// A reference a policy writes as input.meta.scan is the terms input, meta and
// scan, in that order.
const (
	refMetaPos  = 1
	refGroupPos = 2
)

// metaGroupOf classifies one reference: input.meta.<group> names a group,
// input.meta on its own or with a group this cannot read names the whole of
// meta, and anything else is not about meta at all.
func metaGroupOf(ref opaast.Ref) (string, refKind) {
	if len(ref) <= refMetaPos || !ref[0].Equal(opaast.InputRootDocument) {
		return "", refNotMeta
	}

	if name, ok := ref[refMetaPos].Value.(opaast.String); !ok || string(name) != celVarMeta {
		return "", refNotMeta
	}

	if len(ref) <= refGroupPos {
		return "", refWholeMeta
	}

	group, ok := ref[refGroupPos].Value.(opaast.String)
	if !ok {
		// A variable or expression in place of the group, as in
		// input.meta[g], which is only known while the policy runs.
		return "", refWholeMeta
	}

	return string(group), refGroup
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
