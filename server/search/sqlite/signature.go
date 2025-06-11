// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	coretypesv2 "github.com/agntcy/dir/api/core/v1alpha2"
	"gorm.io/gorm"
)

type SignatureObject interface {
	ToSQLiteSignature(agentID uint) (*coretypesv2.SQLiteSignature, error)
}

func (s *SQLiteDB) addSignatureTx(tx *gorm.DB, signature SignatureObject, agentID uint) (uint, error) {
	SQLSignature, err := signature.ToSQLiteSignature(agentID)
	if err != nil {
		return 0, fmt.Errorf("failed to convert signature to SQLite signature: %w", err)
	}

	if err := tx.Create(SQLSignature).Error; err != nil {
		return 0, fmt.Errorf("failed to add signature to SQLite search database: %w", err)
	}

	logger.Info("Added signature to SQLite search database", "agent_id", agentID, "SQLSignature_ID", SQLSignature.ID)

	return SQLSignature.ID, nil
}
