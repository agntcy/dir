// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/server/types"
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

// PolicyFactDependency records that a version of a policy reads one group of a
// record's metadata. A version with no row here reads none of it.
//
// Keyed by version because what a policy reads is part of what a policy is:
// the version is a hash of the content the groups were read from, so the rows
// of one version are settled once written, and an edit arrives as a new
// version with rows of its own.
type PolicyFactDependency struct {
	PolicyID      string `gorm:"column:policy_id;primaryKey;not null"`
	PolicyVersion string `gorm:"column:policy_version;primaryKey;not null"`

	// FactGroup is one of types.FactGroup*.
	FactGroup string `gorm:"column:fact_group;primaryKey;not null"`
}

// RegisterPolicyVersion records that an evaluator is running version of
// policyID now, and which fact groups that version reads. Registering a
// version again only refreshes when it was last seen, which is how a policy
// returned to an earlier version becomes the current one again.
//
// Both are written in one transaction, because a version visible without its
// dependencies reads as a policy that depends on no metadata, and a metadata
// write would then leave its verdicts alone forever. The invalidation query
// distinguishes the two by the version row: registered with no dependency row
// means it reads nothing, and no version row at all means nothing is known
// about it, which is treated as depending on everything.
func (d *DB) RegisterPolicyVersion(policyID, policyVersion string, factGroups []string) error {
	now := time.Now()

	row := &PolicyVersion{PolicyID: policyID, PolicyVersion: policyVersion, FirstSeenAt: now, LastSeenAt: now}

	err := d.gormDB.Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "policy_id"}, {Name: "policy_version"}},
			DoUpdates: clause.AssignmentColumns([]string{"last_seen_at"}),
		}).Create(row).Error
		if err != nil {
			return fmt.Errorf("write version: %w", err)
		}

		return writePolicyFactDependencies(tx, policyID, policyVersion, factGroups)
	})
	if err != nil {
		return fmt.Errorf("register policy version: %w", err)
	}

	return nil
}

// writePolicyFactDependencies records the groups version reads.
//
// A policy that reads nothing still gets a row, carrying types.FactGroupNone.
// Recording that explicitly is what lets the invalidation query tell a policy
// that reads no metadata from one nothing is known about, which it cannot
// skip. Both would otherwise be an empty result.
//
// The rows are inserted and never updated: a version is a hash of the content
// they were read from, so re-registering one can only produce the rows that
// are already there. Nothing is deleted for the same reason — a version whose
// groups appear to have changed is a different version.
func writePolicyFactDependencies(tx *gorm.DB, policyID, policyVersion string, factGroups []string) error {
	if len(factGroups) == 0 {
		factGroups = []string{types.FactGroupNone}
	}

	rows := make([]PolicyFactDependency, 0, len(factGroups))
	for _, group := range factGroups {
		rows = append(rows, PolicyFactDependency{
			PolicyID:      policyID,
			PolicyVersion: policyVersion,
			FactGroup:     group,
		})
	}

	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
		return fmt.Errorf("write fact dependencies: %w", err)
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
