// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package initialize

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/spf13/cobra"
)

func NewCommand(_ *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initialize",
		Short: "Initialize a new agent.json file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitializeCommand()
		},
	}

	return cmd
}

func runInitializeCommand() error {
	agent := coretypes.Agent{}

	// Agent Name
	fmt.Print("Enter agent name: ")
	_, err := fmt.Scanln(&agent.Name)
	if err != nil {
		return fmt.Errorf("failed to read agent name: %w", err)
	}

	// Agent Version
	fmt.Print("Enter agent version: ")
	_, err = fmt.Scanln(&agent.Version)
	if err != nil {
		return fmt.Errorf("failed to read agent version: %w", err)
	}

	// Agent Description
	fmt.Print("Enter description: ")
	_, err = fmt.Scanln(&agent.Description)
	if err != nil {
		return fmt.Errorf("failed to read description: %w", err)
	}

	// Agent Authors
	fmt.Print("Enter author(s) (comma-separated): ")
	var authorsInput string
	_, err = fmt.Scanln(&authorsInput)
	if err != nil {
		return fmt.Errorf("failed to read authors: %w", err)
	}
	agent.Authors = strings.Split(authorsInput, ",")

	// Agent CreatedAt
	agent.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// Agent Skills
	fmt.Print("Enter skill class_uid(s) (comma-separated) (https://schema.oasf.agntcy.org/skillsInput): ")
	var skillsInput string
	_, err = fmt.Scanln(&skillsInput)
	if err != nil {
		return fmt.Errorf("failed to read skills: %w", err)
	}

	classUIDs := strings.Split(authorsInput, ",")
	var skills []*coretypes.Skill
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
	fmt.Print("Enter locator(s) (type1=url1,type2=url2) (https://schema.oasf.agntcy.org/objects/locator): ")
	var locatorsInput string
	_, err = fmt.Scanln(&locatorsInput)
	if err != nil {
		return fmt.Errorf("failed to read locators: %w", err)
	}

	locators := strings.Split(locatorsInput, ",")
	for _, locator := range locators {
		parts := strings.Split(locator, "=")
		if len(parts) != 2 {
			return fmt.Errorf("invalid locator format: %s", locator)
		}

		locatorType := strings.TrimSpace(parts[0])
		locatorURL := strings.TrimSpace(parts[1])

		if locatorType == "" || locatorURL == "" {
			return fmt.Errorf("locator type or URL cannot be empty")
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

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&agent); err != nil {
		return fmt.Errorf("failed to write agent.json: %w", err)
	}

	fmt.Println("agent.json has been successfully created.")
	return nil
}
