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

var _ types.UsageMetricsObject = (*RecordUsageMetrics)(nil)

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

func (m *RecordUsageMetrics) GetRecordCID() string      { return m.RecordCID }
func (m *RecordUsageMetrics) GetPullCount() uint64      { return m.PullCount }
func (m *RecordUsageMetrics) GetProviderCount() uint32  { return m.ProviderCount }
func (m *RecordUsageMetrics) GetExportCount() uint64    { return m.ExportCount }
func (m *RecordUsageMetrics) GetViewCount() uint64      { return m.ViewCount }
func (m *RecordUsageMetrics) GetLastUsedAt() *time.Time { return m.LastUsedAt }

// IncrementPullCount atomically increments pull_count and sets last_used_at.
// Creates the row on first use via an upsert.
func (d *DB) IncrementPullCount(cid string) error {
	now := time.Now().UTC()
	row := &RecordUsageMetrics{
		RecordCID:  cid,
		PullCount:  1,
		LastUsedAt: &now,
	}

	err := d.gormDB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "record_cid"}},
		DoUpdates: clause.Assignments(map[string]any{
			"pull_count":   gorm.Expr("pull_count + 1"),
			"last_used_at": now,
			"updated_at":   now,
		}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("failed to increment pull count for %s: %w", cid, err)
	}

	return nil
}

// SetProviderCount sets the provider_count gauge for a record.
// Creates the row if it does not exist.
func (d *DB) SetProviderCount(cid string, count uint32) error {
	now := time.Now().UTC()
	row := &RecordUsageMetrics{
		RecordCID:     cid,
		ProviderCount: count,
	}

	err := d.gormDB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "record_cid"}},
		DoUpdates: clause.Assignments(map[string]any{
			"provider_count": count,
			"updated_at":     now,
		}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("failed to set provider count for %s: %w", cid, err)
	}

	return nil
}

// GetUsageMetrics returns usage metrics for a record. Returns a zero-value
// result (not an error) when no usage has been recorded yet.
func (d *DB) GetUsageMetrics(cid string) (types.UsageMetricsObject, error) {
	var m RecordUsageMetrics

	err := d.gormDB.Where("record_cid = ?", cid).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &RecordUsageMetrics{RecordCID: cid}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get usage metrics for %s: %w", cid, err)
	}

	return &m, nil
}
