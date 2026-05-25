// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// AI Catalog Agent Finder filter — see Appendix A of the spec.
//
// Grammar (informal):
//
//	filter   = clause { WS+ "AND" WS+ clause } ;
//	clause   = field "=" value ;
//	field    = "displayName" | "type" | "publisherId"
//	         | "createdAfter" | "updatedAfter" ;
//	value    = token { "," token } ;
//	token    = unquoted_token | quoted_string ;
//
// Logical AND across fields. Comma-OR within a single field's value list.
// Each filter field may appear at most once.

// agentFilter is the parsed representation of the listAgents `filter` query.
type agentFilter struct {
	// DisplayName, when non-empty, requests a case-insensitive substring
	// match (LIKE %x%) on the module's display_name column.
	DisplayName string

	// Types is the OR-set of AI Catalog media types the caller is interested
	// in (e.g. "application/a2a-agent-card+json").
	Types []string

	// PublisherIDs is the OR-set of publisher identifiers. Currently unused
	// at the data layer (no publisher column yet); kept here so the parser
	// is spec-complete and ready for future wiring.
	PublisherIDs []string

	// CreatedAfter, when non-zero, filters modules with CreatedAt strictly
	// greater than this timestamp.
	CreatedAfter time.Time

	// UpdatedAfter, when non-zero, filters modules with UpdatedAt strictly
	// greater than this timestamp.
	UpdatedAfter time.Time
}

// parseAgentFilter parses the Agent Finder Specification filter syntax.
//
// The grammar is intentionally narrow: it accepts exactly the five fields
// listed in Appendix A, AND across fields, and comma-OR within a value list.
// Any other syntax (parentheses, OR keywords, unknown fields, duplicate
// fields, missing values) is rejected with a precise error so callers can
// map it to INVALID_ARGUMENT.
//
// Input length is bounded by the proto validator (max_len=2048) and re-checked
// here to defend the controller-internal path. Empty input is a valid no-op.
func parseAgentFilter(input string) (agentFilter, error) {
	const maxInputLen = 2048

	if len(input) > maxInputLen {
		return agentFilter{}, fmt.Errorf("filter expression too long (%d > %d)", len(input), maxInputLen)
	}

	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return agentFilter{}, nil
	}

	clauses, err := splitFilterClauses(trimmed)
	if err != nil {
		return agentFilter{}, err
	}

	var out agentFilter

	seen := map[string]struct{}{}

	for _, clause := range clauses {
		field, rawValue, ok := splitAtFirstEqual(clause)
		if !ok {
			return agentFilter{}, fmt.Errorf("filter clause %q must contain '='", clause)
		}

		field = strings.TrimSpace(field)
		rawValue = strings.TrimSpace(rawValue)

		if field == "" {
			return agentFilter{}, fmt.Errorf("filter clause %q has empty field name", clause)
		}

		if rawValue == "" {
			return agentFilter{}, fmt.Errorf("filter field %q has empty value", field)
		}

		if _, dup := seen[field]; dup {
			return agentFilter{}, fmt.Errorf("filter field %q appears more than once", field)
		}

		seen[field] = struct{}{}

		values, err := parseValueList(rawValue)
		if err != nil {
			return agentFilter{}, fmt.Errorf("filter field %q: %w", field, err)
		}

		if err := applyClause(&out, field, values); err != nil {
			return agentFilter{}, err
		}
	}

	return out, nil
}

// splitFilterClauses splits the filter on the case-sensitive AND keyword
// that appears between whitespace, while respecting double-quoted tokens.
// Per the spec, AND is the only logical connector; lowercase "and" and "OR"
// are not recognized.
func splitFilterClauses(s string) ([]string, error) {
	var (
		clauses []string
		current strings.Builder
		inQuote bool
	)

	flush := func() {
		text := strings.TrimSpace(current.String())
		if text != "" {
			clauses = append(clauses, text)
		}

		current.Reset()
	}

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if r == '"' {
			inQuote = !inQuote

			current.WriteRune(r)

			continue
		}

		if !inQuote && unicode.IsSpace(r) {
			// Look for the literal "AND" surrounded by whitespace.
			if matchKeywordAt(runes, i+countLeadingSpaces(runes[i:]), "AND") {
				flush()
				// Skip past " AND " (with possibly more leading/trailing spaces).
				i += countLeadingSpaces(runes[i:]) + len("AND") - 1

				continue
			}
		}

		current.WriteRune(r)
	}

	if inQuote {
		return nil, errors.New("unterminated quoted string in filter")
	}

	flush()

	if len(clauses) == 0 {
		return nil, errors.New("filter contained no clauses")
	}

	return clauses, nil
}

// countLeadingSpaces counts whitespace runes at the start of r.
func countLeadingSpaces(r []rune) int {
	n := 0

	for _, c := range r {
		if !unicode.IsSpace(c) {
			break
		}

		n++
	}

	return n
}

// matchKeywordAt reports whether the rune slice starting at idx exactly
// matches keyword followed by a whitespace boundary (or end of input).
func matchKeywordAt(r []rune, idx int, keyword string) bool {
	kw := []rune(keyword)
	if idx+len(kw) > len(r) {
		return false
	}

	for i, c := range kw {
		if r[idx+i] != c {
			return false
		}
	}

	end := idx + len(kw)
	if end == len(r) {
		return true
	}

	return unicode.IsSpace(r[end])
}

// splitAtFirstEqual returns (left, right, true) on the first '=' outside of
// a quoted string, or ("", "", false) if no '=' is present.
func splitAtFirstEqual(s string) (string, string, bool) {
	inQuote := false

	for i, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == '=' && !inQuote:
			return s[:i], s[i+1:], true
		}
	}

	return "", "", false
}

// parseValueList splits "a,b,\"c, d\"" into ["a", "b", "c, d"].
// Quoted values may contain commas, whitespace, '=', and the literal
// keyword "AND"; unquoted values may NOT contain '=' (it would have been
// consumed as a clause delimiter) and are trimmed of leading/trailing
// whitespace.
//
// Note: the spec uses uppercase "AND" as the clause separator. A lowercase
// "and" sequence inside an unquoted value is allowed (it's just part of
// the value); the surrounding clause splitter rejects misformed input via
// the stricter no-bare-'=' rule below.
//
//nolint:cyclop
func parseValueList(s string) ([]string, error) {
	var (
		out     []string
		current strings.Builder
		inQuote bool
	)

	flush := func() error {
		token := current.String()
		if !inQuote {
			token = strings.TrimSpace(token)
		}

		if token == "" {
			return errors.New("empty value in value list")
		}

		out = append(out, token)

		current.Reset()

		return nil
	}

	for i := range len(s) {
		c := s[i]

		switch {
		case c == '"':
			if inQuote {
				inQuote = false
			} else {
				if current.Len() != 0 && strings.TrimSpace(current.String()) != "" {
					return nil, errors.New("unexpected '\"' inside unquoted token")
				}

				current.Reset()

				inQuote = true
			}
		case c == ',' && !inQuote:
			if err := flush(); err != nil {
				return nil, err
			}
		case c == '=' && !inQuote:
			// An unquoted '=' inside a value is almost always a sign the
			// caller meant to AND-separate two clauses but used a non-keyword
			// separator (e.g. lowercase "and" or missing whitespace). Rather
			// than silently absorbing it into the displayName value, reject
			// with a precise error so the caller can fix their input.
			return nil, errors.New("unexpected '=' in value (did you forget the 'AND' clause separator? quote the value if you intended to include '=')")
		default:
			current.WriteByte(c)
		}
	}

	if inQuote {
		return nil, errors.New("unterminated quoted value")
	}

	if err := flush(); err != nil {
		return nil, err
	}

	return out, nil
}

// applyClause routes a parsed clause into the appropriate agentFilter field.
// Unknown field names are rejected.
func applyClause(out *agentFilter, field string, values []string) error {
	switch field {
	case "displayName":
		if len(values) != 1 {
			return fmt.Errorf("filter field %q accepts a single value, got %d", field, len(values))
		}

		out.DisplayName = values[0]

		return nil

	case "type":
		out.Types = values

		return nil

	case "publisherId":
		out.PublisherIDs = values

		return nil

	case "createdAfter":
		if len(values) != 1 {
			return fmt.Errorf("filter field %q accepts a single value, got %d", field, len(values))
		}

		ts, err := parseRFC3339(values[0])
		if err != nil {
			return fmt.Errorf("filter field %q: %w", field, err)
		}

		out.CreatedAfter = ts

		return nil

	case "updatedAfter":
		if len(values) != 1 {
			return fmt.Errorf("filter field %q accepts a single value, got %d", field, len(values))
		}

		ts, err := parseRFC3339(values[0])
		if err != nil {
			return fmt.Errorf("filter field %q: %w", field, err)
		}

		out.UpdatedAfter = ts

		return nil

	default:
		return fmt.Errorf("unknown filter field %q (allowed: displayName, type, publisherId, createdAfter, updatedAfter)", field)
	}
}

// parseRFC3339 accepts both RFC3339 and RFC3339Nano (which the standard
// library parses with the same time.RFC3339 layout via the trailing nanos
// path) — matches the ISO 8601 wording in Appendix A.
func parseRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid ISO 8601 timestamp %q: %w", s, err)
	}

	return t.UTC(), nil
}
