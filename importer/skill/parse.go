// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package skill parses Agent Skills directories (https://agentskills.io/specification).
package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
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

// ParseSkillDirectory reads skillDir/SKILL.md, validates frontmatter and directory name, and returns
// a structpb payload for OASF translation. Contract (for oasf-sdk SkillMarkdownToRecord):
//   - skillMarkdown: full SKILL.md content (required string)
//
// Additional normalized fields are included for importer metadata/dedup convenience.
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

	body = strings.TrimSpace(body)

	fields, err := skillPayloadFields(&fm, absDir, body, string(raw))
	if err != nil {
		return nil, err
	}

	st, err := structpb.NewStruct(fields)
	if err != nil {
		return nil, fmt.Errorf("skill payload struct: %w", err)
	}

	return st, nil
}

func skillPayloadFields(fm *skillFrontmatter, absDir, body, skillMarkdown string) (map[string]any, error) {
	fields := map[string]any{
		"skillMarkdown": skillMarkdown,
		"name":          fm.Name,
		"description":   fm.Description,
		"body":          body,
		"skill_root":    absDir,
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
