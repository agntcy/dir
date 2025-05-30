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

func (s *SQLiteDB) GetManyAgentsByFilters(filters searchtypes.QueryFilters) ([]*coretypes.Agent, error) {
	var dbAgents []searchtypes.Agent
	query := s.gormDB.Model(&searchtypes.Agent{})

	// Apply basic filters
	if filters.Name != "" {
		query = query.Where("agents.name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Version != "" {
		query = query.Where("agents.version = ?", filters.Version)
	}
	if filters.Description != "" {
		query = query.Where("agents.description LIKE ?", "%"+filters.Description+"%")
	}

	// Filter by authors
	if len(filters.Authors) > 0 {
		for _, author := range filters.Authors {
			query = query.Where("agents.authors LIKE ?", "%"+author+"%")
		}
	}

	// Filter by skills
	if len(filters.SkillNames) > 0 || len(filters.SkillCategories) > 0 {
		query = query.Joins("JOIN skills ON skills.agent_id = agents.id")

		if len(filters.SkillNames) > 0 {
			query = query.Where("skills.class_name IN ?", filters.SkillNames)
		}

		if len(filters.SkillCategories) > 0 {
			query = query.Where("skills.category_name IN ?", filters.SkillCategories)
		}
	}

	// Filter by locator types
	if len(filters.LocatorTypes) > 0 {
		query = query.Joins("JOIN locators ON locators.agent_id = agents.id")
		query = query.Where("locators.type IN ?", filters.LocatorTypes)
	}

	// Filter by extension names and versions
	if len(filters.ExtensionNames) > 0 || len(filters.ExtensionVersions) > 0 {
		query = query.Joins("JOIN extensions ON extensions.agent_id = agents.id")

		if len(filters.ExtensionNames) > 0 {
			query = query.Where("extensions.name IN ?", filters.ExtensionNames)
		}

		if len(filters.ExtensionVersions) > 0 {
			query = query.Where("extensions.version IN ?", filters.ExtensionVersions)
		}
	}

	// Make query distinct when using joins to avoid duplicate agents
	if len(filters.SkillNames) > 0 || len(filters.SkillCategories) > 0 ||
		len(filters.LocatorTypes) > 0 || len(filters.ExtensionNames) > 0 ||
		len(filters.ExtensionVersions) > 0 {
		query = query.Distinct()
	}

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Preload all related entities to avoid N+1 queries
	query = query.Preload("Skills").
		Preload("Locators").
		Preload("Extensions").
		Preload("Signature")

	// Execute the query with all preloaded relations
	if err := query.Find(&dbAgents).Error; err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}

	coreAgents, agents, err := convertToCoreAgents(dbAgents)
	if err != nil {
		return agents, fmt.Errorf("failed to convert agents: %w", err)
	}

	return coreAgents, nil
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

func convertToCoreAgents(dbAgents []searchtypes.Agent) ([]*coretypes.Agent, error) {
	// Convert DB agents to core agents
	coreAgents := make([]*coretypes.Agent, 0, len(dbAgents))
	for _, dbAgent := range dbAgents {
		// Convert the agent
		coreAgent, err := dbAgent.ToCoreAgent()
		if err != nil {
			return nil, fmt.Errorf("failed to convert DB agent to core agent: %w", err)
		}

		// Convert preloaded skills
		coreSkills := make([]*coretypes.Skill, 0, len(dbAgent.Skills))
		for _, skill := range dbAgent.Skills {
			coreSkill, err := skill.ToCoreSkill()
			if err != nil {
				return nil, fmt.Errorf("failed to convert skill: %w", err)
			}
			coreSkills = append(coreSkills, coreSkill)
		}
		coreAgent.Skills = coreSkills

		// Convert preloaded extensions
		coreExtensions := make([]*coretypes.Extension, 0, len(dbAgent.Extensions))
		for _, ext := range dbAgent.Extensions {
			coreExt, err := ext.ToCoreExtension()
			if err != nil {
				return nil, fmt.Errorf("failed to convert extension: %w", err)
			}
			coreExtensions = append(coreExtensions, coreExt)
		}
		coreAgent.Extensions = coreExtensions

		// Convert preloaded locators
		coreLocators := make([]*coretypes.Locator, 0, len(dbAgent.Locators))
		for _, locator := range dbAgent.Locators {
			coreLocator, err := locator.ToCoreLocator()
			if err != nil {
				return nil, fmt.Errorf("failed to convert locator: %w", err)
			}
			coreLocators = append(coreLocators, coreLocator)
		}
		coreAgent.Locators = coreLocators

		// Convert signature
		coreAgent.Signature = dbAgent.Signature.ToCoreSignature()

		coreAgents = append(coreAgents, coreAgent)
	}
	return coreAgents, nil
}
