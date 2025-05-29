// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	"gorm.io/gorm"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	searchtypes "github.com/agntcy/dir/server/search/types"
)

func (s *SQLiteDB) AddAgent(agent *coretypes.Agent) error {
	err := s.gormDB.Transaction(func(tx *gorm.DB) error {
		id, err := s.AddAgentTx(tx, agent)
		if err != nil {
			return fmt.Errorf("failed to add agent transaction: %w", err)
		}

		_, err = s.AddSignatureTx(tx, agent.Signature, id)
		if err != nil {
			return fmt.Errorf("failed to add signature transaction: %w", err)
		}

		for _, extension := range agent.GetExtensions() {
			if _, err = s.AddExtensionTx(tx, extension, id); err != nil {
				return fmt.Errorf("failed to add extension to search index: %w", err)
			}
		}

		for _, locator := range agent.GetLocators() {
			if _, err = s.AddLocatorTx(tx, locator, id); err != nil {
				return fmt.Errorf("failed to add locator to search index: %w", err)
			}
		}

		for _, skill := range agent.GetSkills() {
			if _, err = s.AddSkillTx(tx, skill, id); err != nil {
				return fmt.Errorf("failed to add skill to search index: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to add agent: %w", err)
	}

	return nil
}

func (s *SQLiteDB) AddAgentTx(tx *gorm.DB, coreAgent *coretypes.Agent) (uint, error) {
	agent := &searchtypes.Agent{}
	if err := agent.FromCoreAgent(coreAgent); err != nil {
		return 0, fmt.Errorf("failed to convert core agent to search agent: %w", err)
	}

	if err := tx.Create(agent).Error; err != nil {
		return 0, fmt.Errorf("failed to add agent to SQLite search database: %w", err)
	}

	logger.Info("Added agent to SQLite search database", "name", agent.Name)

	return agent.ID, nil
}
