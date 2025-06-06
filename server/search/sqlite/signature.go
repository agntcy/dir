// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	searchtypes "github.com/agntcy/dir/server/search/types"
	"gorm.io/gorm"
)

func (s *SQLiteDB) AddSignatureTx(tx *gorm.DB, coreSignature *coretypes.Signature, agentID uint) (uint, error) {
	signature := &searchtypes.Signature{}
	if err := signature.FromCoreSignature(coreSignature, agentID); err != nil {
		return 0, fmt.Errorf("failed to convert core signature to search signature: %w", err)
	}

	if err := tx.Create(signature).Error; err != nil {
		return 0, fmt.Errorf("failed to add signature to SQLite search database: %w", err)
	}

	logger.Info("Added signature to SQLite search database", "agentID", agentID)

	return signature.ID, nil
}
