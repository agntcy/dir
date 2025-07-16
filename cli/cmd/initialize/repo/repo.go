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

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "repo",
	Short: "Initialize a new agent.json file",
	Long: `This command initializes a new agent.json file for an agent. It prompts the user for various details about
the agent.

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)
	agent := coretypes.Agent{
		Agent: &objectsv1.Agent{
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	err := setName(cmd, reader, &agent)
	if err != nil {
		return fmt.Errorf("failed to set agent name: %w", err)
	}

	err = setVersion(cmd, &agent)
	if err != nil {
		return fmt.Errorf("failed to set agent version: %w", err)
	}

	err = setDescription(cmd, reader, &agent)
	if err != nil {
		return fmt.Errorf("failed to set agent description: %w", err)
	}

	err = setAuthors(cmd, reader, &agent)
	if err != nil {
		return fmt.Errorf("failed to set agent authors: %w", err)
	}

	err = setSkills(cmd, reader, &agent)
	if err != nil {
		return fmt.Errorf("failed to set agent skills: %w", err)
	}

	err = setLocators(cmd, reader, &agent)
	if err != nil {
		return fmt.Errorf("failed to set agent locators: %w", err)
	}

	err = writeFile(cmd, &agent)
	if err != nil {
		return fmt.Errorf("failed to write agent.json: %w", err)
	}

	return nil
}

func setName(cmd *cobra.Command, reader *bufio.Reader, agent *coretypes.Agent) error {
	presenter.Print(cmd, "Agent name (agntcy/sample-agent): ")

	name, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read agent name: %w", err)
	}

	agent.Name = strings.TrimSpace(name)

	return nil
}

func setVersion(cmd *cobra.Command, agent *coretypes.Agent) error {
	presenter.Print(cmd, "Agent version (v1.0.0): ")

	_, err := fmt.Scanln(&agent.Version)
	if err != nil {
		return fmt.Errorf("failed to read agent version: %w", err)
	}

	return nil
}

func setDescription(cmd *cobra.Command, reader *bufio.Reader, agent *coretypes.Agent) error {
	presenter.Print(cmd, "Description (Research agent for a specific task): ")

	description, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read description: %w", err)
	}

	agent.Description = strings.TrimSpace(description)

	return nil
}

func setAuthors(cmd *cobra.Command, reader *bufio.Reader, agent *coretypes.Agent) error {
	presenter.Print(cmd, "Enter authors (John Doe, Jane Doe): ")

	authorsInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read authors: %w", err)
	}

	for _, author := range strings.Split(authorsInput, ",") {
		trimmedAuthor := strings.TrimSpace(author)
		if trimmedAuthor == "" {
			return errors.New("author name cannot be empty")
		}

		agent.Authors = append(agent.Authors, trimmedAuthor)
	}

	return nil
}

func setSkills(cmd *cobra.Command, reader *bufio.Reader, agent *coretypes.Agent) error {
	presenter.Print(cmd, "Enter skill class_uid(s), https://schema.oasf.agntcy.org/skills (50204,10205): ")

	skillsInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read skills: %w", err)
	}

	classUIDs := strings.Split(skillsInput, ",")
	skills := make([]*objectsv1.Skill, 0, len(classUIDs))

	for _, UIDString := range classUIDs {
		UID, err := strconv.ParseUint(strings.TrimSpace(UIDString), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse class_uid %s: %w", UIDString, err)
		}

		skills = append(skills, &objectsv1.Skill{
			ClassUid: UID,
		})
	}

	agent.Skills = skills

	return nil
}

func setLocators(cmd *cobra.Command, reader *bufio.Reader, agent *coretypes.Agent) error {
	presenter.Print(cmd, "Enter locator(s), https://schema.oasf.agntcy.org/objects/locator (docker-image=<link>,source-code=<link>): ")

	locatorsInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read locators: %w", err)
	}

	for _, locator := range strings.Split(locatorsInput, ",") {
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

		agent.Locators = append(agent.Locators, &objectsv1.Locator{
			Type: locatorType,
			Url:  locatorURL,
		})
	}

	return nil
}

func writeFile(cmd *cobra.Command, agent *coretypes.Agent) error {
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
