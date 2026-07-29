// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"fmt"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"gorm.io/gorm"
)

// moduleBackfillRow is a minimal modules-table projection for migration 002.
type moduleBackfillRow struct {
	ID                uint `gorm:"primarykey"`
	Name              string
	ArtifactMediaType string         `gorm:"column:artifact_media_type"`
	Data              map[string]any `gorm:"column:data;serializer:json"`
}

func (moduleBackfillRow) TableName() string { return "modules" }

func init() {
	register(Migration{
		ID:      "002_backfill_artifact_media_type",
		Details: "Backfill modules.artifact_media_type for Agent Skills from module data artifacts.",
		Run:     runBackfillArtifactMediaType,
	})
}

func runBackfillArtifactMediaType(db *gorm.DB) error {
	if !db.Migrator().HasTable("modules") {
		return nil
	}

	if !db.Migrator().HasColumn(&moduleBackfillRow{}, "ArtifactMediaType") {
		if err := db.Migrator().AddColumn(&moduleBackfillRow{}, "ArtifactMediaType"); err != nil {
			return fmt.Errorf("add modules.artifact_media_type column: %w", err)
		}
	}

	var modules []moduleBackfillRow

	err := db.Model(&moduleBackfillRow{}).
		Where("name = ?", catalogv1.AgentSkillsModuleName).
		Where("artifact_media_type = '' OR artifact_media_type IS NULL").
		Find(&modules).Error
	if err != nil {
		return fmt.Errorf("list Agent Skills modules for backfill: %w", err)
	}

	for _, module := range modules {
		mediaType := catalogv1.ProtocolAgentSkillsMdMediaType
		if moduleHasArtifacts(module.Data) {
			mediaType = catalogv1.ProtocolAgentSkillsBundleMediaType
		}

		if err := db.Model(&moduleBackfillRow{}).
			Where("id = ?", module.ID).
			Update("artifact_media_type", mediaType).Error; err != nil {
			return fmt.Errorf("update module %d artifact_media_type: %w", module.ID, err)
		}
	}

	return nil
}

// moduleHasArtifacts reports whether module data carries a non-empty artifacts array.
// This matches the historical md vs gzip heuristic used before artifact_media_type existed.
func moduleHasArtifacts(data map[string]any) bool {
	if len(data) == 0 {
		return false
	}

	raw, ok := data["artifacts"]
	if !ok || raw == nil {
		return false
	}

	items, ok := raw.([]any)
	if !ok {
		return false
	}

	return len(items) > 0
}
