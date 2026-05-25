// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// AI Catalog Agent Finder paging + ordering helpers (§7.2 of the spec).

const (
	// agentListDefaultPageSize is applied when the request omits page_size
	// or sets it to zero.
	agentListDefaultPageSize uint32 = 20

	// agentListMaxPageSize bounds the page size per the spec ("Max results
	// (default: 20, max: 100)"). Values above this are clamped, not
	// rejected, matching Google AIP-158 semantics.
	agentListMaxPageSize uint32 = 100

	// agentListMaxOrderByClauses bounds the number of comma-separated sort
	// fields. The spec is silent on a limit; cap at a small value to
	// avoid pathological inputs.
	agentListMaxOrderByClauses = 8

	// agentPageTokenMaxLen bounds the on-wire page-token length. We emit
	// short tokens (base64 of a decimal offset), so anything longer than
	// this is invalid by construction.
	agentPageTokenMaxLen = 64
)

// clampPageSize implements "default 20, max 100" with input clamping.
func clampPageSize(requested uint32) uint32 {
	switch {
	case requested == 0:
		return agentListDefaultPageSize
	case requested > agentListMaxPageSize:
		return agentListMaxPageSize
	default:
		return requested
	}
}

// orderByClause describes a single sort directive parsed from order_by.
type orderByClause struct {
	// Column is the validated SQL column identifier (NOT user-controlled).
	Column string
	// Desc is true for DESC, false for ASC.
	Desc bool
}

// orderByColumnAllowList enumerates the sortable columns exposed by the
// Agent Finder API.
var orderByColumnAllowList = map[string]string{
	"name":         "name",
	"display_name": "name",
	"displayname":  "name",
	"created_at":   "oasf_created_at",
	"createdat":    "oasf_created_at",
	"updated_at":   "oasf_created_at",
	"updatedat":    "oasf_created_at",
}

// parseOrderBy parses the order_by query string, e.g. "name, created_at
// DESC". An empty input yields the controller's default ordering
// (created_at DESC).
func parseOrderBy(input string) ([]orderByClause, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return defaultAgentOrder(), nil
	}

	parts := strings.Split(trimmed, ",")
	if len(parts) > agentListMaxOrderByClauses {
		return nil, fmt.Errorf("order_by contains %d clauses; maximum is %d", len(parts), agentListMaxOrderByClauses)
	}

	clauses := make([]orderByClause, 0, len(parts))
	seen := map[string]struct{}{}

	for _, raw := range parts {
		clause := strings.TrimSpace(raw)
		if clause == "" {
			return nil, errors.New("empty order_by clause")
		}

		tokens := strings.Fields(clause)

		var (
			field string
			desc  bool
		)

		switch len(tokens) {
		case 1:
			field = tokens[0]
		case 2: //nolint:mnd
			field = tokens[0]

			switch strings.ToUpper(tokens[1]) {
			case "ASC":
				desc = false
			case "DESC":
				desc = true
			default:
				return nil, fmt.Errorf("invalid order_by direction %q (expected ASC or DESC)", tokens[1])
			}
		default:
			return nil, fmt.Errorf("invalid order_by clause %q", clause)
		}

		col, ok := orderByColumnAllowList[strings.ToLower(field)]
		if !ok {
			return nil, fmt.Errorf("unknown order_by field %q", field)
		}

		if _, dup := seen[col]; dup {
			return nil, fmt.Errorf("order_by field %q appears more than once", field)
		}

		seen[col] = struct{}{}
		clauses = append(clauses, orderByClause{Column: col, Desc: desc})
	}

	return clauses, nil
}

// defaultAgentOrder is applied when order_by is omitted. The spec is
// silent on a default; we pick "oasf_created_at DESC" so the most
// recently created agents appear first. Column is the post-allow-list
// internal name (see orderByColumnAllowList).
func defaultAgentOrder() []orderByClause {
	return []orderByClause{{Column: "oasf_created_at", Desc: true}}
}

// encodePageToken serialises a non-negative offset into the opaque
// continuation token returned in ListAgentsResponse.next_page_token.
// Offset 0 yields an empty token (callers omit next_page_token at the
// end of a result set).
func encodePageToken(offset int) string {
	if offset <= 0 {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// decodePageToken parses a previously issued token. Empty input is
// valid and yields offset 0. Malformed tokens are rejected with a
// precise error so the controller maps them to INVALID_ARGUMENT.
func decodePageToken(s string) (int, error) {
	if s == "" {
		return 0, nil
	}

	if len(s) > agentPageTokenMaxLen {
		return 0, fmt.Errorf("page_token too long (%d > %d)", len(s), agentPageTokenMaxLen)
	}

	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return 0, fmt.Errorf("invalid page_token encoding: %w", err)
	}

	offset, err := strconv.Atoi(string(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid page_token payload: %w", err)
	}

	if offset < 0 {
		return 0, errors.New("page_token offset must be non-negative")
	}

	return offset, nil
}
