// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import "time"

// RecordUsageMetrics tracks per-record usage counters used to derive the
// popularity sort mode. It is a 1:1 companion to the records table, keyed on
// record_cid. Rows are created on first use and deleted alongside the parent
// record via RemoveRecord.
//
// PullCount, ExportCount, and ViewCount are cumulative event counters.
// ProviderCount is a point-in-time gauge refreshed by the reconciler.
type RecordUsageMetrics struct {
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RecordCID     string     `gorm:"column:record_cid;primarykey;not null"`
	PullCount     uint64     `gorm:"column:pull_count;default:0;not null"`
	ProviderCount uint32     `gorm:"column:provider_count;default:0;not null"`
	ExportCount   uint64     `gorm:"column:export_count;default:0;not null"`
	ViewCount     uint64     `gorm:"column:view_count;default:0;not null"`
	LastUsedAt    *time.Time `gorm:"column:last_used_at"`
}
