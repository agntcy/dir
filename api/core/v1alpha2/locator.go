// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha2

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

type SQLiteLocator struct {
	gorm.Model
	AgentID     uint   `gorm:"not null;index"`
	Type        string `gorm:"not null"`
	URL         string `gorm:"not null"`
	Size        uint64 `gorm:"not null"`
	Digest      string `gorm:"not null"`
	Annotations string `gorm:"not null"`
}

func (s *Locator) ToSQLiteLocator(agentID uint) (*SQLiteLocator, error) {
	if s == nil {
		return nil, nil
	}

	locator := &SQLiteLocator{
		AgentID: agentID,
		Type:    s.Type,
		URL:     s.Url,
		Size:    *s.Size,
		Digest:  *s.Digest,
	}

	if s.Annotations != nil {
		annotations, err := json.Marshal(s.Annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotations: %w", err)
		}

		locator.Annotations = string(annotations)
	}

	return locator, nil
}
