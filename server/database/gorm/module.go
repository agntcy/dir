// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
)

type Module struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	RecordCID string `gorm:"column:record_cid;not null;index"`
	Name      string `gorm:"not null"`
	ModuleID  uint64 `gorm:"column:module_id"`

	// ArtifactMediaType is the OASF module.artifact.media_type value when present
	// (e.g. application/agent-skills+gzip).
	ArtifactMediaType string `gorm:"column:artifact_media_type;index"`

	// Data is the module's structured payload (e.g. the A2A agent card or
	// MCP server definition).
	//
	// TODO: this is a temporary solution. The module data will be stored as a
	// separate artifact in the future and will need to be retrieved differently.
	Data map[string]any `gorm:"column:data;serializer:json"`

	// Artifact is the AI Catalog document (urls, json, or data). Only an
	// integration/ai_catalog module stores it; other modules leave it empty.
	Artifact *coretypes.ModuleArtifact `gorm:"column:artifact;serializer:json"`
}

func (module *Module) GetName() string {
	return module.Name
}

func (module *Module) GetID() uint64 {
	return module.ModuleID
}

func (module *Module) GetData() map[string]any {
	return module.Data
}

func (module *Module) GetAnnotations() map[string]string {
	// Modules do not store annotations in the database
	return nil
}

func (module *Module) GetArtifactMediaType() string {
	return module.ArtifactMediaType
}

func (module *Module) GetArtifact() *coretypes.ModuleArtifact {
	return module.Artifact
}

// convertModules transforms interface types to Database structs.
func convertModules(modules []coretypes.Module, recordCID string) []Module {
	result := make([]Module, len(modules))
	for i, module := range modules {
		var artifact *coretypes.ModuleArtifact
		if module.GetName() == catalogv1.AICatalogModuleName {
			artifact = module.GetArtifact()
		}

		result[i] = Module{
			RecordCID:         recordCID,
			Name:              module.GetName(),
			ModuleID:          module.GetID(),
			Data:              module.GetData(),
			ArtifactMediaType: module.GetArtifactMediaType(),
			Artifact:          artifact,
		}
	}

	return result
}
