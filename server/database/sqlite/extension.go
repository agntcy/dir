// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"time"

	"github.com/agntcy/dir/server/types"
)

type Extension struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	RecordCID string `gorm:"column:record_cid;not null;index"`
	Name      string `gorm:"not null"`
	Version   string `gorm:"not null"`
}

func (extension *Extension) GetAnnotations() map[string]string {
	// SQLite extensions don't store annotations, return empty map
	return make(map[string]string)
}

func (extension *Extension) GetName() string {
	return extension.Name
}

func (extension *Extension) GetVersion() string {
	return extension.Version
}

func (extension *Extension) GetData() map[string]any {
	// SQLite extensions don't store data, return empty map
	return make(map[string]any)
}

// convertExtensions transforms interface types to SQLite structs.
func convertExtensions(extensions []types.Extension, recordCID string) []Extension {
	result := make([]Extension, len(extensions))
	for i, extension := range extensions {
		result[i] = Extension{
			RecordCID: recordCID,
			Name:      extension.GetName(),
			Version:   extension.GetVersion(),
		}
	}

	return result
}
