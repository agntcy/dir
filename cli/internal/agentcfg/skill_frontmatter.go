// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"fmt"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// frontmatter holds the SKILL.md frontmatter fields the renderers reuse.
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// splitFrontmatter separates a leading YAML frontmatter block (delimited by ---)
// from the document body. A document without frontmatter returns an empty
// frontmatter and the whole document as the body.
func splitFrontmatter(doc string) (frontmatter, string, error) {
	const delim = "---"

	normalized := strings.ReplaceAll(doc, "\r\n", "\n")
	if !strings.HasPrefix(normalized, delim+"\n") {
		return frontmatter{}, doc, nil
	}

	rest := normalized[len(delim)+1:]

	fmText, body, found := strings.Cut(rest, "\n"+delim)
	if !found {
		return frontmatter{}, doc, nil
	}

	body = strings.TrimLeft(body, "\n")

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return frontmatter{}, "", fmt.Errorf("parse skill frontmatter: %w", err)
	}

	return fm, body, nil
}
