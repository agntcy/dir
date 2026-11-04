// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"strings"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/policy/predicate"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

// applyPolicyGate keeps only rows whose record complies with every enforced
// policy, when searches enforce them: for each, the record needs an
// evaluated, compliant verdict at the version currently in force. cidColumn
// is the column holding the record's CID in the queried table, e.g.
// "records.record_cid" or "skills.record_cid". In shadow mode searches are not
// filtered; CountRecordsExcluded tells what would be.
//
// The gate fails closed. It requires a positive verdict rather than
// excluding negative ones, so a record with no verdict yet, a failed
// evaluation, or only a verdict under a superseded version is excluded too.
// It is applied by the query builder itself, so no caller-supplied filter
// can lift it.
func (d *DB) applyPolicyGate(query *gorm.DB, cidColumn string) *gorm.DB {
	enforcement := d.currentEnforcement()

	if enforcement.Search == policyconfig.ModeEnforce {
		query = gatePolicies(query, cidColumn, enforcement.Policies)
	}

	return gateSearchPolicies(query, cidColumn, enforcement.EnforcingSearchPolicies())
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

// IsRecordServable reports whether the record with cid may be returned by a
// read that fetches it directly rather than searching. Unless fetches check
// the enforced policies every record may be, indexed or not. Otherwise only a
// record that passes the policy gate may be, so a record not yet indexed may
// not, and the observer is told of each one that may not. In shadow mode that
// is all that happens: every record is still served, even when the check
// fails.
func (d *DB) IsRecordServable(cid string) (bool, error) {
	// Read once: deciding the mode and then gating on a second read could see
	// two different settings.
	enforcement := d.currentEnforcement()
	if !enforcement.Fetch.Active() || len(enforcement.Policies) == 0 {
		return true, nil
	}

	shadow := enforcement.Fetch == policyconfig.ModeShadow

	compliant, err := d.IsRecordCompliant(cid, enforcement.Policies, nil)
	if err != nil {
		if shadow {
			logger.Warn("Shadow mode: could not check record against enforced policies", "cid", cid, "error", err)

			return true, nil
		}

		return false, err
	}

	if compliant {
		return true, nil
	}

	if d.observer != nil {
		d.observer.FetchExcluded(enforcement.Fetch)
	}

	if shadow {
		logger.Debug("Shadow mode: enforced policies would exclude record", "cid", cid)

		return true, nil
	}

	return false, nil
}

// CountRecordsExcluded counts the indexed records a search applying policies
// and search would leave out.
func (d *DB) CountRecordsExcluded(policies []types.EnforcedPolicy, search []types.SearchPolicy) (int64, error) {
	var count int64

	if err := d.recordsExcluded(policies, search).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count records excluded by enforced policies: %w", err)
	}

	return count, nil
}

// ListRecordsExcluded returns the CIDs of the indexed records a search
// applying policies and search would leave out, in CID order, skipping offset
// and returning at most limit of them; a limit of zero returns them all.
func (d *DB) ListRecordsExcluded(policies []types.EnforcedPolicy, search []types.SearchPolicy, limit, offset int) ([]string, error) {
	query := d.recordsExcluded(policies, search).Order("records.record_cid").Offset(offset)
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
// policies and search: the complement of what that gate serves, built by the
// same two calls applyPolicyGate makes so that the answer cannot drift from
// what queries actually do.
func (d *DB) recordsExcluded(policies []types.EnforcedPolicy, search []types.SearchPolicy) *gorm.DB {
	served := d.gormDB.Model(&Record{}).Select("records.record_cid")
	served = gatePolicies(served, "records.record_cid", policies)
	served = gateSearchPolicies(served, "records.record_cid", search)

	return d.gormDB.Model(&Record{}).Where("records.record_cid NOT IN (?)", served)
}

// IsRecordCompliant reports whether the record with cid is indexed and
// satisfies both policies and search, whatever this view enforces.
//
// A fetch passes no search policies: one of those decides what a query
// returns, not whether a record may be read by CID.
func (d *DB) IsRecordCompliant(cid string, policies []types.EnforcedPolicy, search []types.SearchPolicy) (bool, error) {
	var count int64

	query := d.gormDB.Model(&Record{}).Where("records.record_cid = ?", cid)
	query = gateSearchPolicies(gatePolicies(query, "records.record_cid", policies), "records.record_cid", search)

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

// gatePolicies requires, for each policy, an evaluated, compliant verdict at
// its version for the record in cidColumn. See applyPolicyGate.
func gatePolicies(query *gorm.DB, cidColumn string, policies []types.EnforcedPolicy) *gorm.DB {
	for _, policy := range policies {
		query = query.Where(verdictExists(cidColumn), verdictArgs(policy.ID, policy.Version)...)
	}

	return query
}

// gatePredicate keeps only rows whose record satisfies p, reading the version
// in force for each policy p names from policies.
//
// The counterpart to gatePolicies for a policy that is a boolean combination
// rather than a conjunction. A disjunction cannot be chained as successive
// Where calls, which only ever narrow, so this renders one condition and
// applies it once.
func gatePredicate(query *gorm.DB, cidColumn string, p predicate.Node, policies []types.EnforcedPolicy) (*gorm.DB, error) {
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
