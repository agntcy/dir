// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	register(Migration{
		ID:      "003_record_fk_on_delete_cascade",
		Details: "Set ON DELETE CASCADE on name_verifications, scan_reports, and record_usage_metrics FKs to records.",
		Run:     runRecordFKOnDeleteCascade,
	})
}

type RecordStub struct {
	RecordCID string `gorm:"column:record_cid;primaryKey;not null"`

	ScanReports      []ScanReportStub        `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	NameVerification *NameVerificationStub   `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
	UsageMetrics     *RecordUsageMetricsStub `gorm:"foreignKey:RecordCID;references:RecordCID;constraint:OnDelete:CASCADE"`
}

func (RecordStub) TableName() string { return "records" }

type ScanReportStub struct {
	RecordCID string `gorm:"column:record_cid;primaryKey;not null"`
}

func (ScanReportStub) TableName() string { return "scan_reports" }

type NameVerificationStub struct {
	RecordCID string `gorm:"column:record_cid;not null;uniqueIndex"`
}

func (NameVerificationStub) TableName() string { return "name_verifications" }

type RecordUsageMetricsStub struct {
	RecordCID string `gorm:"column:record_cid;primaryKey;not null"`
}

func (RecordUsageMetricsStub) TableName() string { return "record_usage_metrics" }

var associations = []struct {
	relation string
	child    any
}{
	{relation: "ScanReports", child: &ScanReportStub{}},
	{relation: "NameVerification", child: &NameVerificationStub{}},
	{relation: "UsageMetrics", child: &RecordUsageMetricsStub{}},
}

func runRecordFKOnDeleteCascade(db *gorm.DB) error {
	if !db.Migrator().HasTable(&RecordStub{}) {
		return nil
	}

	migrator := db.Migrator()

	for _, assoc := range associations {
		if !migrator.HasTable(assoc.child) {
			continue
		}

		if migrator.HasConstraint(&RecordStub{}, assoc.relation) {
			if err := migrator.DropConstraint(&RecordStub{}, assoc.relation); err != nil {
				return fmt.Errorf("drop %s FK constraint: %w", assoc.relation, err)
			}
		}

		if err := migrator.CreateConstraint(&RecordStub{}, assoc.relation); err != nil {
			return fmt.Errorf("create %s FK constraint: %w", assoc.relation, err)
		}
	}

	return nil
}
