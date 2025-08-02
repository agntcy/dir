// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"time"

	"github.com/agntcy/dir/server/types"
)

type Locator struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	RecordCID string `gorm:"column:record_cid;not null;index"`
	Type      string `gorm:"not null"`
	URL       string `gorm:"not null"`
}

func (locator *Locator) GetAnnotations() map[string]string {
	// SQLite locators don't store annotations, return empty map
	return make(map[string]string)
}

func (locator *Locator) GetType() string {
	return locator.Type
}

func (locator *Locator) GetURL() string {
	return locator.URL
}

func (locator *Locator) GetSize() uint64 {
	// SQLite locators don't store size information
	return 0
}

func (locator *Locator) GetDigest() string {
	// SQLite locators don't store digest information
	return ""
}

// convertLocators transforms interface types to SQLite structs.
func convertLocators(locators []types.Locator, recordCID string) []Locator {
	result := make([]Locator, len(locators))
	for i, locator := range locators {
		result[i] = Locator{
			RecordCID: recordCID,
			Type:      locator.GetType(),
			URL:       locator.GetURL(),
		}
	}

	return result
}
