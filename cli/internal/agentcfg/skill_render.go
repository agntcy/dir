// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"strconv"
	"strings"
)

// kv is an ordered frontmatter key/value pair. Value is emitted verbatim, so
// callers quote strings as needed.
type kv struct {
	key   string
	value string
}

// withFrontmatter renders a YAML frontmatter block followed by the body.
func withFrontmatter(pairs []kv, body string) []byte {
	var b strings.Builder

	b.WriteString("---\n")

	for _, p := range pairs {
		b.WriteString(p.key)
		b.WriteString(": ")
		b.WriteString(p.value)
		b.WriteString("\n")
	}

	b.WriteString("---\n\n")
	b.WriteString(body)

	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}

	return []byte(b.String())
}

// renderCursor renders a Cursor project rule (.mdc): description + alwaysApply.
func renderCursor(canonical string) ([]byte, error) {
	fm, body, err := splitFrontmatter(canonical)
	if err != nil {
		return nil, err
	}

	return withFrontmatter([]kv{
		{"description", strconv.Quote(fm.Description)},
		{"alwaysApply", "true"},
	}, body), nil
}

// renderCopilot renders a Copilot instructions file: applyTo glob.
func renderCopilot(canonical string) ([]byte, error) {
	_, body, err := splitFrontmatter(canonical)
	if err != nil {
		return nil, err
	}

	return withFrontmatter([]kv{
		{"applyTo", strconv.Quote("**")},
	}, body), nil
}

// renderContinue renders a Continue rule: name + description + alwaysApply.
func renderContinue(canonical string) ([]byte, error) {
	fm, body, err := splitFrontmatter(canonical)
	if err != nil {
		return nil, err
	}

	return withFrontmatter([]kv{
		{"name", strconv.Quote(fm.Name)},
		{"description", strconv.Quote(fm.Description)},
		{"alwaysApply", "true"},
	}, body), nil
}

// renderRoo renders a Roo rule: body only, frontmatter stripped.
func renderRoo(canonical string) ([]byte, error) {
	return renderBodyOnly(canonical)
}

// renderCline renders a Cline rule: plain markdown body, no frontmatter.
func renderCline(canonical string) ([]byte, error) {
	return renderBodyOnly(canonical)
}

// renderManagedInner renders the inner content for a managed block: body only.
func renderManagedInner(canonical string) ([]byte, error) {
	return renderBodyOnly(canonical)
}

func renderBodyOnly(canonical string) ([]byte, error) {
	_, body, err := splitFrontmatter(canonical)
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}

	return []byte(body), nil
}
