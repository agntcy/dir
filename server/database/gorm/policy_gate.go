// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"

	policyconfig "github.com/agntcy/dir/server/policy/config"
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
	if enforcement.Search != policyconfig.ModeEnforce {
		return query
	}

	return gatePolicies(query, cidColumn, enforcement.Policies)
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

	compliant, err := d.IsRecordCompliant(cid, enforcement.Policies)
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

// CountRecordsExcluded counts the indexed records that do not comply with
// every one of policies: those an enforcing search excludes.
func (d *DB) CountRecordsExcluded(policies []types.EnforcedPolicy) (int64, error) {
	var count int64

	if err := d.recordsExcluded(policies).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count records excluded by enforced policies: %w", err)
	}

	return count, nil
}

// ListRecordsExcluded returns the CIDs of the indexed records that do not
// comply with every one of policies, in CID order, skipping offset and
// returning at most limit of them; a limit of zero returns them all.
func (d *DB) ListRecordsExcluded(policies []types.EnforcedPolicy, limit, offset int) ([]string, error) {
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

// recordsExcluded selects the indexed records that do not comply with every
// one of policies.
func (d *DB) recordsExcluded(policies []types.EnforcedPolicy) *gorm.DB {
	compliant := gatePolicies(d.gormDB.Model(&Record{}).Select("records.record_cid"), "records.record_cid", policies)

	return d.gormDB.Model(&Record{}).Where("records.record_cid NOT IN (?)", compliant)
}

// IsRecordCompliant reports whether the record with cid is indexed and
// complies with every one of policies, whatever this view enforces.
func (d *DB) IsRecordCompliant(cid string, policies []types.EnforcedPolicy) (bool, error) {
	var count int64

	query := d.gormDB.Model(&Record{}).Where("records.record_cid = ?", cid)
	if err := gatePolicies(query, "records.record_cid", policies).Count(&count).Error; err != nil {
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
		query = query.Where(
			"EXISTS (SELECT 1 FROM policy_evaluations pe WHERE pe.record_cid = "+cidColumn+
				" AND pe.policy_id = ? AND pe.policy_version = ? AND pe.status IN ? AND pe.compliant = ?)",
			policy.ID, policy.Version, types.EvaluatedPolicyStatuses(), true,
		)
	}

	return query
}
