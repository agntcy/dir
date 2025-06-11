// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha2

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

type SQLiteRecord struct {
	gorm.Model
	Name        string `gorm:"not null"`
	Version     string `gorm:"not null"`
	Description string `gorm:"not null"`
	Authors     string `gorm:"not null"`
	Annotations string `gorm:"not null"`
	Tags        string `gorm:"not null"`

	Skills     []Skill     `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
	Locators   []Locator   `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
	Extensions []Extension `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
	Signature  Signature   `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
}

func (s *Record) ToSQLiteRecord() (*SQLiteRecord, error) {
	if s == nil {
		return nil, nil
	}

	record := &SQLiteRecord{
		Name:        s.Name,
		Version:     s.Version,
		Description: s.Description,
	}

	if s.Authors != nil {
		authors, err := json.Marshal(s.Authors)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal authors: %w", err)
		}

		record.Authors = string(authors)
	}

	if s.Annotations != nil {
		annotations, err := json.Marshal(s.Annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotations: %w", err)
		}

		record.Annotations = string(annotations)
	}

	if s.Tags != nil {
		tags, err := json.Marshal(s.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}

		record.Tags = string(tags)
	}

	return record, nil
}
