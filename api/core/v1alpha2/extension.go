// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha2

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

type SQLiteExtension struct {
	gorm.Model
	AgentID     uint   `gorm:"not null;index"`
	Name        string `gorm:"not null"`
	Version     string `gorm:"not null"`
	Data        string `gorm:"not null"`
	Annotations string `gorm:"not null"`
}

func (s *Extension) ToSQLiteExtension(agentID uint) (*SQLiteExtension, error) {
	if s == nil {
		return nil, nil
	}

	extension := &SQLiteExtension{
		AgentID: agentID,
		Name:    s.Name,
		Version: *s.Version,
		Data:    s.ExtensionData.String(),
	}

	if s.Annotations != nil {
		annotations, err := json.Marshal(s.Annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotations: %w", err)
		}

		extension.Annotations = string(annotations)
	}

	return extension, nil
}
