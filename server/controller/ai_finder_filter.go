// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/translator"
)

// AI Finder filter grammar:
//
//	filter = clause { WS+ "AND" WS+ clause } ;
//	clause = field "=" value ;
//	field  = "displayName" | "type" | "publisherId" | "createdAfter" | "updatedAfter" ;
//	value  = token { "," token } ;
//	token  = unquoted_token | quoted_string ;
//
// Logical AND across fields, comma-OR within a field's value list; each field
// may appear at most once.

// filterMaxLen mirrors the proto validator (max_len=2048).
const filterMaxLen = 2048

// agentFilter is the parsed representation of the ListAgents filter query.
type agentFilter struct {
	DisplayName  string
	Types        []string
	PublisherIDs []string
	CreatedAfter time.Time
	UpdatedAfter time.Time
}

// oasfModuleForMediaType maps a media type onto the OASF module name the
// registry indexes, or ("", false) for unknown types.
func oasfModuleForMediaType(mediaType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case catalogv1.ProtocolA2ACardJsonMediaType:
		return translator.A2AModuleName, true
	case catalogv1.ProtocolMCPCardJsonMediaType:
		return translator.MCPModuleName, true
	case catalogv1.ProtocolAgentSkillsMdMediaType:
		return translator.AgentSkillsModuleName, true
	case catalogv1.ProtocolAgentSkillsBundleMediaType:
		return translator.AgentSkillsModuleName, true
	default:
		return "", false
	}
}

// filterCatalogEntriesByMediaType keeps entries whose media_type matches one of
// the requested AI Catalog types. Required for Agent Skills because markdown
// and bundle records share the same OASF module name in the DB index.
func filterCatalogEntriesByMediaType(entries []*catalogv1.CatalogEntry, mediaTypes []string) []*catalogv1.CatalogEntry {
	if len(mediaTypes) == 0 || len(entries) == 0 {
		return entries
	}

	allowed := make(map[string]struct{}, len(mediaTypes))
	for _, mt := range mediaTypes {
		allowed[strings.ToLower(strings.TrimSpace(mt))] = struct{}{}
	}

	out := make([]*catalogv1.CatalogEntry, 0, len(entries))
	for _, entry := range entries {
		if _, ok := allowed[strings.ToLower(entry.GetMediaType())]; ok {
			out = append(out, entry)
		}
	}

	return out
}
// (parentheses, OR keywords, unknown or duplicate fields, missing values) is
// rejected so callers map it to INVALID_ARGUMENT. Empty input is a no-op.
func parseAgentFilter(input string) (agentFilter, error) {
	if len(input) > filterMaxLen {
		return agentFilter{}, fmt.Errorf("filter expression too long (%d > %d)", len(input), filterMaxLen)
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

// buildRecordFilterOptions translates a parsed filter, order, and paging into
// FilterOptions for the catalog query layer. The bool is false when type= was
// set but no requested media type maps to an indexed module (zero rows).
func buildRecordFilterOptions(f agentFilter, order []orderByClause, pageSize, offset int) ([]types.FilterOption, bool) {
	opts := []types.FilterOption{
		types.WithLimit(pageSize),
		types.WithOffset(offset),
	}

	if f.DisplayName != "" {
		opts = append(opts, types.WithNames("*"+f.DisplayName+"*"))
	}

	if len(f.Types) > 0 {
		var modules []string

		for _, mt := range f.Types {
			if module, ok := oasfModuleForMediaType(mt); ok {
				modules = append(modules, module)
			}
		}

		if len(modules) == 0 {
			return nil, false
		}

		opts = append(opts, types.WithModuleNames(modules...))
	}

	// createdAfter / updatedAfter both resolve to a strict '>' comparison on
	// the only OASF-supplied record timestamp.
	if !f.CreatedAfter.IsZero() {
		opts = append(opts, types.WithCreatedAts(">"+f.CreatedAfter.UTC().Format(time.RFC3339)))
	}

	if !f.UpdatedAfter.IsZero() {
		opts = append(opts, types.WithCreatedAts(">"+f.UpdatedAfter.UTC().Format(time.RFC3339)))
	}

	if len(order) > 0 {
		clauses := make([]types.RecordOrderClause, 0, len(order))
		for _, o := range order {
			clauses = append(clauses, types.RecordOrderClause{Column: o.Column, Desc: o.Desc})
		}

		opts = append(opts, types.WithOrderBy(clauses...))
	}

	return opts, true
}

// splitFilterClauses splits on the case-sensitive "AND" keyword between
// whitespace, respecting double-quoted tokens. AND is the only connector.
func splitFilterClauses(s string) ([]string, error) {
	var (
		clauses []string
		current strings.Builder
		inQuote bool
	)

	flush := func() {
		if text := strings.TrimSpace(current.String()); text != "" {
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

		if !inQuote && unicode.IsSpace(r) && matchKeywordAt(runes, i+countLeadingSpaces(runes[i:]), "AND") {
			flush()

			i += countLeadingSpaces(runes[i:]) + len("AND") - 1

			continue
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

// matchKeywordAt reports whether the runes at idx match keyword followed by a
// whitespace boundary or end of input.
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

// splitAtFirstEqual returns (left, right, true) on the first '=' outside a
// quoted string, or ("", "", false) when no '=' is present.
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

// parseValueList splits "a,b,\"c, d\"" into ["a", "b", "c, d"]. Quoted values
// may contain commas, whitespace and '='; unquoted values may not contain '='
// and are trimmed.
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
			return nil, errors.New("unexpected '=' in value (quote the value if it is intended)")
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

// applyClause routes a parsed clause into the matching agentFilter field.
//
//nolint:cyclop
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
		ts, err := singleTimestamp(field, values)
		if err != nil {
			return err
		}

		out.CreatedAfter = ts

		return nil

	case "updatedAfter":
		ts, err := singleTimestamp(field, values)
		if err != nil {
			return err
		}

		out.UpdatedAfter = ts

		return nil

	default:
		return fmt.Errorf("unknown filter field %q (allowed: displayName, type, publisherId, createdAfter, updatedAfter)", field)
	}
}

// singleTimestamp validates a single-value RFC3339 clause.
func singleTimestamp(field string, values []string) (time.Time, error) {
	if len(values) != 1 {
		return time.Time{}, fmt.Errorf("filter field %q accepts a single value, got %d", field, len(values))
	}

	ts, err := time.Parse(time.RFC3339, values[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("filter field %q: invalid ISO 8601 timestamp %q: %w", field, values[0], err)
	}

	return ts.UTC(), nil
}
