// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

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
}

func (module *Module) GetName() string {
	return module.Name
}

func (module *Module) GetData() map[string]any {
	// SQLite modules don't store data, return empty map
	return make(map[string]any)
}

// convertModules transforms interface types to SQLite structs.
func convertModules(modules []types.Module, recordCID string) []Module {
	result := make([]Module, len(modules))
	for i, module := range modules {
		result[i] = Module{
			RecordCID: recordCID,
			Name:      module.GetName(),
		}
	}

	return result
}
