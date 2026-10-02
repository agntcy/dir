// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

// applyPolicyGate keeps only rows whose record complies with every enforced
// policy: for each, the record needs an evaluated, compliant verdict at the
// version currently in force. cidColumn is the column holding the record's
// CID in the queried table, e.g. "records.record_cid" or "skills.record_cid".
//
// The gate fails closed. It requires a positive verdict rather than
// excluding negative ones, so a record with no verdict yet, a failed
// evaluation, or only a verdict under a superseded version is excluded too.
// It is applied by the query builder itself, so no caller-supplied filter
// can lift it.
func (d *DB) applyPolicyGate(query *gorm.DB, cidColumn string) *gorm.DB {
	return gatePolicies(query, cidColumn, d.currentEnforcedPolicies())
}

// IsRecordServable reports whether the record with cid may be returned by a
// read that fetches it directly rather than searching. With no policy
// enforced every record may be, indexed or not. With policies enforced only
// a record that passes the policy gate may be, so a record not yet indexed
// may not.
func (d *DB) IsRecordServable(cid string) (bool, error) {
	// Read once: deciding "nothing is enforced" and then gating on a second
	// read could see two different sets.
	policies := d.currentEnforcedPolicies()
	if len(policies) == 0 {
		return true, nil
	}

	var count int64

	query := d.gormDB.Model(&Record{}).Where("records.record_cid = ?", cid)
	if err := gatePolicies(query, "records.record_cid", policies).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check record against enforced policies: %w", err)
	}

	return count > 0, nil
}

// currentEnforcedPolicies returns the enforced set for one query.
func (d *DB) currentEnforcedPolicies() []types.EnforcedPolicy {
	if d.enforcedPolicies == nil {
		return nil
	}

	return d.enforcedPolicies()
}

// gatePolicies requires, for each policy, an evaluated, compliant verdict at
// its version for the record in cidColumn. See applyPolicyGate.
func gatePolicies(query *gorm.DB, cidColumn string, policies []types.EnforcedPolicy) *gorm.DB {
	for _, policy := range policies {
		query = query.Where(
			"EXISTS (SELECT 1 FROM policy_evaluations pe WHERE pe.record_cid = "+cidColumn+
				" AND pe.policy_id = ? AND pe.policy_version = ? AND pe.status IN ? AND pe.compliant = ?)",
			policy.ID, policy.Version, types.EvaluatedPolicyStatuses(), true,
		)
	}

	return query
}
