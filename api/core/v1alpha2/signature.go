// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha2

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

type SQLiteSignature struct {
	gorm.Model
	AgentID       uint   `gorm:"not null;index"`
	Annotations   string `gorm:"not null"`
	SignedAt      string `gorm:"not null"`
	Algorithm     string `gorm:"not null"`
	Signature     string `gorm:"not null"`
	Certificate   string `gorm:"not null"`
	ContentType   string `gorm:"not null"`
	ContentBundle string `gorm:"not null"`
}

func (s *Signature) ToSQLiteSignature(agentID uint) (*SQLiteSignature, error) {
	if s == nil {
		return nil, nil
	}

	signature := &SQLiteSignature{
		AgentID:       agentID,
		SignedAt:      s.SignedAt,
		Algorithm:     s.Algorithm,
		Signature:     s.Signature,
		Certificate:   s.Certificate,
		ContentType:   s.ContentType,
		ContentBundle: s.ContentBundle,
	}

	if s.Annotations != nil {
		annotations, err := json.Marshal(s.Annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotations: %w", err)
		}

		signature.Annotations = string(annotations)
	}

	return signature, nil
}
