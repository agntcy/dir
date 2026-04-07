// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package skill parses Agent Skills directories (https://agentskills.io/specification).
package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
)

const (
	maxSkillNameLen        = 64
	maxSkillDescriptionLen = 1024
	maxCompatibilityLen    = 500
)

// skillFrontmatter captures YAML frontmatter from SKILL.md (Agent Skills spec).
type skillFrontmatter struct {
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description"`
	License       string         `yaml:"license"`
	Compatibility string         `yaml:"compatibility"`
	AllowedTools  string         `yaml:"allowed-tools"`
	Metadata      map[string]any `yaml:"metadata"`
}

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ParseSkillDirectory reads skillDir/SKILL.md, validates frontmatter and directory name, and returns
// a structpb payload for OASF translation. Contract (for oasf-sdk AgentSkillToRecord):
//   - name, description, body (required strings)
//   - license, compatibility, skill_root (optional strings)
//   - allowed_tools: list of strings (from frontmatter allowed-tools)
//   - metadata: struct of string values (YAML metadata map; may include version)
func ParseSkillDirectory(skillDir string) (*structpb.Struct, error) {
	absDir, err := filepath.Abs(skillDir)
	if err != nil {
		return nil, fmt.Errorf("resolve skill directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("stat skill directory: %w", err)
	}

	if !info.IsDir() {
		return nil, errors.New("skill path must be a directory")
	}

	skillPath := filepath.Join(absDir, "SKILL.md")

	raw, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("missing SKILL.md in %s", absDir)
		}

		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}

	fmYAML, body, err := splitSkillFrontmatter(string(raw))
	if err != nil {
		return nil, err
	}

	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(fmYAML), &fm); err != nil {
		return nil, fmt.Errorf("parse SKILL.md frontmatter: %w", err)
	}

	fm.Name = strings.TrimSpace(fm.Name)
	fm.Description = strings.TrimSpace(fm.Description)
	fm.License = strings.TrimSpace(fm.License)
	fm.Compatibility = strings.TrimSpace(fm.Compatibility)
	fm.AllowedTools = strings.TrimSpace(fm.AllowedTools)

	if err := validateSkillFrontmatter(&fm, filepath.Base(absDir)); err != nil {
		return nil, err
	}

	body = strings.TrimSpace(body)

	fields, err := skillPayloadFields(&fm, absDir, body)
	if err != nil {
		return nil, err
	}

	st, err := structpb.NewStruct(fields)
	if err != nil {
		return nil, fmt.Errorf("skill payload struct: %w", err)
	}

	return st, nil
}

func skillPayloadFields(fm *skillFrontmatter, absDir, body string) (map[string]any, error) {
	fields := map[string]any{
		"name":        fm.Name,
		"description": fm.Description,
		"body":        body,
		"skill_root":  absDir,
	}

	if fm.License != "" {
		fields["license"] = fm.License
	}

	if fm.Compatibility != "" {
		fields["compatibility"] = fm.Compatibility
	}

	if fm.AllowedTools != "" {
		tools := strings.Fields(fm.AllowedTools)
		if len(tools) > 0 {
			allowed := make([]any, len(tools))
			for i, t := range tools {
				allowed[i] = t
			}

			fields["allowed_tools"] = allowed
		}
	}

	if len(fm.Metadata) > 0 {
		metaStr := make(map[string]any, len(fm.Metadata))
		for k, v := range fm.Metadata {
			metaStr[k] = fmt.Sprint(v)
		}

		metaStruct, err := structpb.NewStruct(metaStr)
		if err != nil {
			return nil, fmt.Errorf("metadata as struct: %w", err)
		}

		fields["metadata"] = metaStruct.AsMap()
	}

	return fields, nil
}

func splitSkillFrontmatter(raw string) (string, string, error) {
	raw = strings.TrimPrefix(raw, "\ufeff")

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", errors.New("SKILL.md is empty")
	}

	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return "", "", errors.New("SKILL.md must start with YAML frontmatter (---)")
	}

	end := -1

	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i

			break
		}
	}

	if end < 0 {
		return "", "", errors.New("SKILL.md frontmatter must end with ---")
	}

	fmLines := lines[1:end]
	bodyLines := lines[end+1:]

	return strings.Join(fmLines, "\n"), strings.Join(bodyLines, "\n"), nil
}

func validateSkillFrontmatter(fm *skillFrontmatter, dirBase string) error {
	if fm.Name == "" {
		return errors.New("SKILL.md frontmatter: name is required")
	}

	if fm.Description == "" {
		return errors.New("SKILL.md frontmatter: description is required")
	}

	n := utf8.RuneCountInString(fm.Name)
	if n < 1 || n > maxSkillNameLen {
		return fmt.Errorf("SKILL.md name must be 1–%d characters", maxSkillNameLen)
	}

	if !skillNamePattern.MatchString(fm.Name) {
		return errors.New("SKILL.md name must be lowercase alphanumeric with single hyphens (no leading/trailing hyphen, no --)")
	}

	if strings.Contains(fm.Name, "--") {
		return errors.New("SKILL.md name must not contain consecutive hyphens")
	}

	if d := utf8.RuneCountInString(fm.Description); d < 1 || d > maxSkillDescriptionLen {
		return fmt.Errorf("SKILL.md description must be 1–%d characters", maxSkillDescriptionLen)
	}

	if fm.Name != dirBase {
		return fmt.Errorf("SKILL.md name %q must match directory name %q", fm.Name, dirBase)
	}

	if fm.Compatibility != "" {
		if c := utf8.RuneCountInString(fm.Compatibility); c > maxCompatibilityLen {
			return fmt.Errorf("SKILL.md compatibility exceeds %d characters", maxCompatibilityLen)
		}
	}

	return nil
}
