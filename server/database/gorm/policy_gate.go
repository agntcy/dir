// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/server/policy/predicate"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

// applyPolicyGate keeps only rows the search policies being enforced admit.
// cidColumn is the column holding the record's CID in the queried table, e.g.
// "records.record_cid" or "skills.record_cid". A policy in shadow mode does
// not filter; CountRecordsExcluded tells what it would.
//
// The gate fails closed. A policy's term requires a positive verdict rather
// than excluding negative ones, so a record with no verdict yet, a failed
// evaluation, or only a verdict under a superseded version is excluded too.
// It is applied by the query builder itself, so no caller-supplied filter
// can lift it.
func (d *DB) applyPolicyGate(query *gorm.DB, cidColumn string) *gorm.DB {
	return gateSearchPolicies(query, cidColumn, d.currentEnforcement().EnforcingSearchPolicies())
}

// gateSearchPolicies applies each search policy's predicate to the query.
//
// Every policy given is applied, whatever its mode says: choosing the set is
// the caller's, because the counters ask what a set other than the enforcing
// one would exclude.
//
// A policy that cannot be rendered fails the query rather than being left
// out. The error reaches the caller through the query itself, which is what
// keeps this applicable from a function that returns no error and stops the
// query from running at all: a read that silently dropped a policy would
// return records nobody decided to serve.
func gateSearchPolicies(query *gorm.DB, cidColumn string, policies []types.SearchPolicy) *gorm.DB {
	for _, policy := range policies {
		gated, err := gatePredicate(query, cidColumn, policy.Predicate, policy.Reads)
		if err != nil {
			_ = query.AddError(fmt.Errorf("apply search policy %q: %w", policy.ID, err))

			return query
		}

		query = gated
	}

	return query
}

// CountRecordsExcluded counts the indexed records a search applying policies
// would leave out.
func (d *DB) CountRecordsExcluded(policies []types.SearchPolicy) (int64, error) {
	var count int64

	if err := d.recordsExcluded(policies).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count records excluded by enforced policies: %w", err)
	}

	return count, nil
}

// ListRecordsExcluded returns the CIDs of the indexed records a search
// applying policies would leave out, in CID order, skipping offset and
// returning at most limit of them; a limit of zero returns them all.
func (d *DB) ListRecordsExcluded(policies []types.SearchPolicy, limit, offset int) ([]string, error) {
	query := d.recordsExcluded(policies).Order("records.record_cid").Offset(offset)
	if limit > 0 {
		query = query.Limit(limit)
	}

	var cids []string

	if err := query.Pluck("records.record_cid", &cids).Error; err != nil {
		return nil, fmt.Errorf("list records excluded by enforced policies: %w", err)
	}

	return cids, nil
}

// recordsExcluded selects the indexed records left out by a search gated on
// policies: the complement of what that gate serves, built by the same call
// applyPolicyGate makes so that the answer cannot drift from what queries
// actually do.
func (d *DB) recordsExcluded(policies []types.SearchPolicy) *gorm.DB {
	served := gateSearchPolicies(d.gormDB.Model(&Record{}).Select("records.record_cid"), "records.record_cid", policies)

	return d.gormDB.Model(&Record{}).Where("records.record_cid NOT IN (?)", served)
}

// IsRecordCompliant reports whether the record with cid is indexed and
// satisfies policies, whatever this view itself applies.
func (d *DB) IsRecordCompliant(cid string, policies []types.SearchPolicy) (bool, error) {
	var count int64

	query := d.gormDB.Model(&Record{}).Where("records.record_cid = ?", cid)
	query = gateSearchPolicies(query, "records.record_cid", policies)

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("check record against enforced policies: %w", err)
	}

	return count > 0, nil
}

// currentEnforcement returns what one read must satisfy.
func (d *DB) currentEnforcement() types.PolicyEnforcement {
	if d.enforcement == nil {
		return types.PolicyEnforcement{}
	}

	return d.enforcement()
}

// gatePredicate narrows query to the records satisfying p, with each term
// read at the version in policies. A disjunction cannot be chained as
// successive Where calls, which only ever narrow, so the whole tree is
// rendered into one condition and applied once.
func gatePredicate(query *gorm.DB, cidColumn string, p predicate.Node, policies []types.VersionedPolicy) (*gorm.DB, error) {
	versions := make(map[string]string, len(policies))
	for _, policy := range policies {
		versions[policy.ID] = policy.Version
	}

	args := make([]any, 0, len(policies)*verdictArgCount)

	condition, err := renderPredicate(p, cidColumn, versions, &args)
	if err != nil {
		return nil, err
	}

	return query.Where(condition, args...), nil
}

// renderPredicate renders p as one SQL condition, appending its arguments to
// args in the order their placeholders appear.
//
// Composites are parenthesized whatever they contain: the condition is read
// by a database rather than a person, so there is nothing to gain by knowing
// when the parentheses are redundant, and a precedence bug here would be a
// quiet one.
func renderPredicate(p predicate.Node, cidColumn string, versions map[string]string, args *[]any) (string, error) {
	switch node := p.(type) {
	case predicate.Verdict:
		version, inForce := versions[node.PolicyID]
		if !inForce {
			// No version in force means no honest condition to render:
			// omitting the term serves every record and refusing every
			// record hides them all. A policy reading a verdict that is not
			// enforceable is held in shadow and never reaches this, so
			// arriving here is a defect rather than a state to tolerate.
			return "", fmt.Errorf("gate on policy %q, which has no version in force", node.PolicyID)
		}

		*args = append(*args, verdictArgs(node.PolicyID, version)...)

		return verdictExists(cidColumn), nil
	case predicate.All:
		return renderParts(node.Parts, " AND ", cidColumn, versions, args)
	case predicate.Any:
		return renderParts(node.Parts, " OR ", cidColumn, versions, args)
	default:
		return "", fmt.Errorf("gate on unsupported predicate %T", p)
	}
}

// renderParts renders parts joined by operator.
func renderParts(parts []predicate.Node, operator, cidColumn string, versions map[string]string, args *[]any) (string, error) {
	if len(parts) == 0 {
		// Neither reading of an empty composite is safe to guess at, and no
		// expression produces one: CEL's operators are binary.
		return "", errors.New("gate on a predicate combining nothing")
	}

	rendered := make([]string, 0, len(parts))

	for _, part := range parts {
		condition, err := renderPredicate(part, cidColumn, versions, args)
		if err != nil {
			return "", err
		}

		rendered = append(rendered, condition)
	}

	return "(" + strings.Join(rendered, operator) + ")", nil
}

// verdictExists is the condition both gates are built from: the record in
// cidColumn has an evaluated, compliant verdict at the version asked for.
//
// Repeating the pe alias across several of these in one condition is sound.
// They are sibling scopes rather than nested ones, so each resolves to its
// own subquery.
func verdictExists(cidColumn string) string {
	return "EXISTS (SELECT 1 FROM policy_evaluations pe WHERE pe.record_cid = " + cidColumn +
		" AND pe.policy_id = ? AND pe.policy_version = ? AND pe.status IN ? AND pe.compliant = ?)"
}

// verdictArgCount is how many arguments verdictArgs binds.
const verdictArgCount = 4

// verdictArgs binds verdictExists for one policy at one version.
func verdictArgs(policyID, version string) []any {
	return []any{policyID, version, types.EvaluatedPolicyStatuses(), true}
}
