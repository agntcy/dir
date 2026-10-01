// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PolicyBackfill records that a policy's verdicts once covered every indexed
// record, so the policy can be enforced. It is kept once written: records
// indexed later have no verdict until evaluated, which the gate already
// excludes, and must not make the policy pending again.
type PolicyBackfill struct {
	ID uint `gorm:"column:id;primaryKey;autoIncrement"`

	PolicyID     string    `gorm:"column:policy_id;not null;uniqueIndex"`
	BackfilledAt time.Time `gorm:"column:backfilled_at;not null"`
}

// CountRecordsNeedingPolicyEvaluation counts the indexed records with no
// evaluated verdict under policyVersion of policyID: those the reconciler
// still has to evaluate, and that reads exclude until it has.
func (d *DB) CountRecordsNeedingPolicyEvaluation(policyID, policyVersion string) (int64, error) {
	var count int64

	if err := d.needingPolicyEvaluation(policyID, policyVersion).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count records needing policy evaluation: %w", err)
	}

	return count, nil
}

// MarkPolicyBackfilled records that the verdicts of policyID covered the
// indexed records. Marking it again keeps the first time.
func (d *DB) MarkPolicyBackfilled(policyID string) error {
	row := &PolicyBackfill{PolicyID: policyID, BackfilledAt: time.Now()}

	if err := d.gormDB.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
		return fmt.Errorf("mark policy backfilled: %w", err)
	}

	return nil
}

// PolicyBackfilled reports whether the verdicts of policyID were marked as
// covering the indexed records.
func (d *DB) PolicyBackfilled(policyID string) (bool, error) {
	var row PolicyBackfill

	err := d.gormDB.Where("policy_id = ?", policyID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("check policy backfill: %w", err)
	}

	return true, nil
}
