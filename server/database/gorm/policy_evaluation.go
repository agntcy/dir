// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PolicyEvaluation stores one policy verdict per (record_cid, policy_id). The
// verdict carries the version of the policy it was reached under, and a
// re-evaluation under another version replaces it.
//
// This is storage only: no read path filters on it yet, and nothing computes
// it yet. See #2207 (async computation, backfill) and #2208 (unconditional
// exclusion on Search/Pull/Resolve).
type PolicyEvaluation struct {
	ID uint `gorm:"column:id;primaryKey;autoIncrement"`

	RecordCID     string `gorm:"column:record_cid;not null;uniqueIndex:idx_policy_evaluations_record_policy,priority:1"`
	PolicyID      string `gorm:"column:policy_id;not null;uniqueIndex:idx_policy_evaluations_record_policy,priority:2"`
	PolicyVersion string `gorm:"column:policy_version;not null;index"`

	// Compliant is a fail-closed placeholder (false) unless Status is
	// PolicyEvalStatusEvaluated. See types.EvaluatedPolicyStatuses.
	Compliant bool   `gorm:"column:compliant;not null"`
	Status    string `gorm:"column:status;not null;index"` // one of types.PolicyEvalStatus*
	Reason    string `gorm:"column:reason"`

	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;index"`
}

// Ensure PolicyEvaluation implements types.PolicyEvaluationObject.
var _ types.PolicyEvaluationObject = (*PolicyEvaluation)(nil)

func (p *PolicyEvaluation) GetRecordCID() string     { return p.RecordCID }
func (p *PolicyEvaluation) GetPolicyID() string      { return p.PolicyID }
func (p *PolicyEvaluation) GetPolicyVersion() string { return p.PolicyVersion }
func (p *PolicyEvaluation) GetCompliant() bool       { return p.Compliant }
func (p *PolicyEvaluation) GetStatus() string        { return p.Status }
func (p *PolicyEvaluation) GetReason() string        { return p.Reason }
func (p *PolicyEvaluation) GetUpdatedAt() time.Time  { return p.UpdatedAt }

// UpsertPolicyEvaluation inserts or replaces the verdict row keyed by
// (record_cid, policy_id). A failed evaluation is written with compliant =
// false, the same fail-closed placeholder convention as ScanReport.IsSafe on
// a failed scan — see EvaluatedPolicyStatuses.
func (d *DB) UpsertPolicyEvaluation(eval types.PolicyEvaluationObject) error {
	now := time.Now()

	row := &PolicyEvaluation{
		RecordCID:     eval.GetRecordCID(),
		PolicyID:      eval.GetPolicyID(),
		PolicyVersion: eval.GetPolicyVersion(),
		Compliant:     eval.GetCompliant(),
		Status:        eval.GetStatus(),
		Reason:        eval.GetReason(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if row.Status != types.PolicyEvalStatusEvaluated {
		row.Compliant = false
	}

	err := d.gormDB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "record_cid"}, {Name: "policy_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"policy_version",
			"compliant",
			"status",
			"reason",
			"updated_at",
		}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("upsert policy evaluation: %w", err)
	}

	return nil
}

// GetPolicyEvaluations retrieves every policy verdict recorded for a record.
func (d *DB) GetPolicyEvaluations(recordCID string) ([]types.PolicyEvaluationObject, error) {
	var rows []PolicyEvaluation

	if err := d.gormDB.Where("record_cid = ?", recordCID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("get policy evaluations: %w", err)
	}

	result := make([]types.PolicyEvaluationObject, 0, len(rows))
	for i := range rows {
		result = append(result, &rows[i])
	}

	return result, nil
}

// GetRecordsNeedingPolicyEvaluation returns records with no evaluated
// policy_evaluations row for policyID at policyVersion: never evaluated
// against this version, or whose last evaluation failed.
//
// A failed row must not suppress re-selection. It reads as non-compliant
// (fail closed), so if it also stopped the record being retried, one
// transient evaluator error would exclude the record from every read until
// the policy version changed.
//
// Records come in record_cid order, starting after afterCID, at most limit of
// them; a limit of zero returns them all. Paging by the last record_cid seen,
// rather than by offset, costs the same however far in the caller is, and
// does not return a record whose evaluation failed again within one pass.
func (d *DB) GetRecordsNeedingPolicyEvaluation(policyID, policyVersion, afterCID string, limit int) ([]coretypes.Record, error) {
	var records []Record

	query := d.needingPolicyEvaluation(policyID, policyVersion).
		Where("records.record_cid > ?", afterCID).
		Order("records.record_cid")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("get records needing policy evaluation: %w", err)
	}

	result := make([]coretypes.Record, 0, len(records))
	for i := range records {
		result = append(result, &records[i])
	}

	return result, nil
}

// needingPolicyEvaluation selects the indexed records with no evaluated
// verdict under policyVersion of policyID.
func (d *DB) needingPolicyEvaluation(policyID, policyVersion string) *gorm.DB {
	return d.gormDB.Table("records").
		Where(`NOT EXISTS (
			SELECT 1 FROM policy_evaluations pe
			WHERE pe.record_cid = records.record_cid
			AND pe.policy_id = ?
			AND pe.policy_version = ?
			AND pe.status IN ?
		)`, policyID, policyVersion, types.EvaluatedPolicyStatuses())
}
