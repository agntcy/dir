// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

// Owner stores an ownership claim for a record.
type Owner struct {
	RecordCID string `gorm:"column:record_cid;not null;index"`
	OwnerID   string `gorm:"column:owner_id;not null;index"`
	ClaimedAt string `gorm:"column:claimed_at"`
}
