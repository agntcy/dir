// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"time"

	"github.com/agntcy/dir/server/types"
)

type Domain struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	RecordCID string `gorm:"column:record_cid;not null;index"`
	DomainID  uint64 `gorm:"not null"`
	Name      string `gorm:"not null"`
}

func (domain *Domain) GetAnnotations() map[string]string {
	// SQLite domains don't store annotations, return empty map
	return make(map[string]string)
}

func (domain *Domain) GetName() string {
	return domain.Name
}

func (domain *Domain) GetID() uint64 {
	return domain.DomainID
}

// convertDomains converts domain interfaces to SQLite Domain structs.
func convertDomains(domains []types.Domain, recordCID string) []Domain {
	result := make([]Domain, len(domains))
	for i, domain := range domains {
		result[i] = Domain{
			RecordCID: recordCID,
			DomainID:  domain.GetID(),
			Name:      domain.GetName(),
		}
	}

	return result
}
