// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"time"

	"github.com/agntcy/dir/server/types"
)

type Module struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	RecordCID string `gorm:"column:record_cid;not null;index"`
	Name      string `gorm:"not null"`
	ModuleID  uint64 `gorm:"column:module_id"`

	// AI Catalog required fields
	// TODO: AI Catalog either requires a url or a data field, as currently we save the module data, we will only set the data field and leave the url field empty.
	// TODO: Display name should be the retrieved from the module's data
	// TODO: Tags should be the retrieved from the module's data
	DisplayName  string         `gorm:"column:display_name"`
	ArtifactURL  string         `gorm:"column:artifact_url"`
	ArtifactData map[string]any `gorm:"column:artifact_data;serializer:json"`
	Tags         string         `gorm:"column:tags"`
}

func (module *Module) GetName() string {
	return module.Name
}

func (module *Module) GetID() uint64 {
	return module.ModuleID
}

func (module *Module) GetData() map[string]any {
	// Database modules don't store data, return empty map
	return make(map[string]any)
}

// convertModules transforms interface types to Database structs.
func convertModules(modules []types.Module, recordCID string, displayName string) []Module {
	result := make([]Module, len(modules))
	for i, module := range modules {
		result[i] = Module{
			RecordCID:    recordCID,
			Name:         module.GetName(),
			ModuleID:     module.GetID(),
			DisplayName:  displayName, // TODO: Display name should be the retrieved from the module's data
			ArtifactData: module.GetData(),
		}
	}

	return result
}
