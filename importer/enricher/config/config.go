// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
)

// Default values for enricher configuration.
const (
	DefaultConfigFile        = "importer/enricher/enricher.json"
	DefaultRequestsPerMinute = 2 // Default maximum LLM API requests per minute (to avoid rate limit errors)
)

//go:embed enricher.skills.prompt.md
var DefaultSkillsPromptTemplate string

//go:embed enricher.domains.prompt.md
var DefaultDomainsPromptTemplate string

// Config contains configuration for the enricher pipeline stage.
type Config struct {
	ConfigFile            string // Path to enricher JSON (model, mcpServers, max-steps)
	SkillsPromptTemplate  string // Path to custom skills prompt template file
	DomainsPromptTemplate string // Path to custom domains prompt template file
	RequestsPerMinute     int    // Maximum LLM API requests per minute (to avoid rate limit errors)
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.ConfigFile == "" {
		return errors.New("config file is required")
	}

	if _, err := os.Stat(c.ConfigFile); err != nil {
		return fmt.Errorf("config file not found: %w", err)
	}

	if c.SkillsPromptTemplate != "" {
		if _, err := os.Stat(c.SkillsPromptTemplate); err != nil {
			return fmt.Errorf("skills prompt template file not found: %w", err)
		}

		data, err := os.ReadFile(c.SkillsPromptTemplate)
		if err != nil {
			return fmt.Errorf("failed to read skills prompt template file %s: %w", c.SkillsPromptTemplate, err)
		}

		c.SkillsPromptTemplate = string(data)
	} else {
		c.SkillsPromptTemplate = DefaultSkillsPromptTemplate
	}

	if c.DomainsPromptTemplate != "" {
		if _, err := os.Stat(c.DomainsPromptTemplate); err != nil {
			return fmt.Errorf("domains prompt template file not found: %w", err)
		}

		data, err := os.ReadFile(c.DomainsPromptTemplate)
		if err != nil {
			return fmt.Errorf("failed to read domains prompt template file %s: %w", c.DomainsPromptTemplate, err)
		}

		c.DomainsPromptTemplate = string(data)
	} else {
		c.DomainsPromptTemplate = DefaultDomainsPromptTemplate
	}

	if c.RequestsPerMinute <= 0 {
		return errors.New("requests per minute must be greater than 0")
	}

	return nil
}
