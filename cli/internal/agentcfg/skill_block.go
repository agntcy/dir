// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import "strings"

func blockBegin(slug string) string {
	return "<!-- BEGIN " + slug + " (managed by dirctl) -->"
}

func blockEnd(slug string) string {
	return "<!-- END " + slug + " -->"
}

// hasBlock reports whether the content already contains our managed block for slug.
func hasBlock(slug, content string) bool {
	return strings.Contains(content, blockBegin(slug)) && strings.Contains(content, blockEnd(slug))
}

// upsertBlock inserts our managed block carrying inner, or replaces the existing
// block for slug in place, preserving all other content.
func upsertBlock(slug, existing, inner string) string {
	begin, end := blockBegin(slug), blockEnd(slug)
	block := begin + "\n" + strings.TrimRight(inner, "\n") + "\n" + end + "\n"

	if hasBlock(slug, existing) {
		before, _, found := strings.Cut(existing, begin)
		if !found {
			return appendBlock(existing, block)
		}

		_, after, found := strings.Cut(existing, end)
		if !found {
			return appendBlock(existing, block)
		}

		after = strings.TrimPrefix(after, "\n")

		return before + block + after
	}

	return appendBlock(existing, block)
}

func appendBlock(existing, block string) string {
	if existing == "" {
		return block
	}

	separator := "\n"
	if strings.HasSuffix(existing, "\n") {
		separator = ""
	}

	return existing + separator + "\n" + block
}

// removeBlock strips our managed block for slug from content. It reports whether
// a block was found and removed.
func removeBlock(slug, content string) (string, bool) {
	if !hasBlock(slug, content) {
		return content, false
	}

	begin, end := blockBegin(slug), blockEnd(slug)

	before, _, found := strings.Cut(content, begin)
	if !found {
		return content, false
	}

	_, after, found := strings.Cut(content, end)
	if !found {
		return content, false
	}

	after = strings.TrimPrefix(after, "\n")
	result := strings.TrimRight(before, "\n")

	if result != "" && after != "" {
		result += "\n"
	}

	return result + after, true
}
