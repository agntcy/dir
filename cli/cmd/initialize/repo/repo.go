// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "repo",
	Short: "Initialize a new agent.json file",
	Long: `

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

// nolint:cyclop
func runCommand(cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)
	agent := coretypes.Agent{}

	// Agent Name
	presenter.Print(cmd, "Enter agent name: ")

	name, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read agent name: %w", err)
	}

	agent.Name = strings.TrimSpace(name)

	// Agent Version
	presenter.Print(cmd, "Enter agent version: ")

	_, err = fmt.Scanln(&agent.Version)
	if err != nil {
		return fmt.Errorf("failed to read agent version: %w", err)
	}

	// Agent Description
	presenter.Print(cmd, "Enter description: ")

	description, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read description: %w", err)
	}

	agent.Description = strings.TrimSpace(description)

	// Agent Authors
	presenter.Print(cmd, "Enter author(s) (comma-separated): ")

	var authorsInput string

	_, err = fmt.Scanln(&authorsInput)
	if err != nil {
		return fmt.Errorf("failed to read authors: %w", err)
	}

	agent.Authors = strings.Split(authorsInput, ",")

	// Agent CreatedAt
	agent.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// Agent Skills
	presenter.Print(cmd, "Enter skill class_uid(s) (comma-separated) (https://schema.oasf.agntcy.org/skills): ")

	var skillsInput string

	_, err = fmt.Scanln(&skillsInput)
	if err != nil {
		return fmt.Errorf("failed to read skills: %w", err)
	}

	classUIDs := strings.Split(skillsInput, ",")
	skills := make([]*coretypes.Skill, 0, len(classUIDs))

	for _, UIDString := range classUIDs {
		UID, err := strconv.ParseUint(UIDString, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse class_uid %s: %w", UIDString, err)
		}

		skills = append(skills, &coretypes.Skill{
			ClassUid: UID,
		})
	}

	agent.Skills = skills

	// Agent Locators
	presenter.Print(cmd, "Enter locator(s) (type1=url1,type2=url2) (https://schema.oasf.agntcy.org/objects/locator): ")

	var locatorsInput string

	_, err = fmt.Scanln(&locatorsInput)
	if err != nil {
		return fmt.Errorf("failed to read locators: %w", err)
	}

	locators := strings.Split(locatorsInput, ",")
	for _, locator := range locators {
		parts := strings.Split(locator, "=")
		//nolint:mnd
		if len(parts) != 2 {
			return fmt.Errorf("invalid locator format: %s", locator)
		}

		locatorType := strings.TrimSpace(parts[0])
		locatorURL := strings.TrimSpace(parts[1])

		if locatorType == "" || locatorURL == "" {
			return errors.New("locator type or URL cannot be empty")
		}

		agent.Locators = append(agent.Locators, &coretypes.Locator{
			Type: locatorType,
			Url:  locatorURL,
		})
	}

	// Write to agent.json
	file, err := os.Create("agent.json")
	if err != nil {
		return fmt.Errorf("failed to create agent.json: %w", err)
	}
	defer file.Close()

	JSONEncoder := json.NewEncoder(file)
	JSONEncoder.SetIndent("", "  ")

	if err := JSONEncoder.Encode(&agent); err != nil {
		return fmt.Errorf("failed to write agent.json: %w", err)
	}

	presenter.Print(cmd, "agent.json file created\n")

	return nil
}
