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

// PolicyVersion records a version of a policy that an evaluator has run.
// Versions come from the policy's content, so editing a policy registers a new
// one and nobody states a version by hand.
type PolicyVersion struct {
	ID uint `gorm:"column:id;primaryKey;autoIncrement"`

	PolicyID      string `gorm:"column:policy_id;not null;uniqueIndex:idx_policy_versions_version,priority:1"`
	PolicyVersion string `gorm:"column:policy_version;not null;uniqueIndex:idx_policy_versions_version,priority:2"`

	FirstSeenAt time.Time `gorm:"column:first_seen_at;not null"`
	LastSeenAt  time.Time `gorm:"column:last_seen_at;not null"`
}

// RegisterPolicyVersion records that an evaluator is running version of
// policyID now. Registering a version again only refreshes when it was last
// seen, which is how a policy returned to an earlier version becomes the
// current one again.
func (d *DB) RegisterPolicyVersion(policyID, policyVersion string) error {
	now := time.Now()

	row := &PolicyVersion{PolicyID: policyID, PolicyVersion: policyVersion, FirstSeenAt: now, LastSeenAt: now}

	err := d.gormDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "policy_id"}, {Name: "policy_version"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_seen_at"}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("register policy version: %w", err)
	}

	return nil
}

// GetCurrentPolicyVersion returns the version of policyID an evaluator ran
// most recently, and whether any evaluator has registered the policy.
func (d *DB) GetCurrentPolicyVersion(policyID string) (string, bool, error) {
	var row PolicyVersion

	err := d.gormDB.Where("policy_id = ?", policyID).Order("last_seen_at DESC, id DESC").Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf("get current policy version: %w", err)
	}

	return row.PolicyVersion, true, nil
}
