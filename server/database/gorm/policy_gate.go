// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
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
	if d.enforcedPolicies == nil {
		return query
	}

	for _, policy := range d.enforcedPolicies() {
		query = query.Where(
			"EXISTS (SELECT 1 FROM policy_evaluations pe WHERE pe.record_cid = "+cidColumn+
				" AND pe.policy_id = ? AND pe.policy_version = ? AND pe.status IN ? AND pe.compliant = ?)",
			policy.ID, policy.Version, types.EvaluatedPolicyStatuses(), true,
		)
	}

	return query
}
