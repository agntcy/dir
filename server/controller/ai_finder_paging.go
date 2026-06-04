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

// AI Finder paging and ordering helpers.

const (
	// agentListDefaultPageSize is applied when page_size is omitted or zero.
	agentListDefaultPageSize uint32 = 20

	// agentListMaxPageSize bounds the page size; larger values are clamped.
	agentListMaxPageSize uint32 = 100

	// agentListMaxOrderByClauses caps the number of sort fields.
	agentListMaxOrderByClauses = 8

	// agentPageTokenMaxLen bounds the on-wire page-token length.
	agentPageTokenMaxLen = 64
)

// clampPageSize applies the default and maximum page size.
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

// orderByClause is a single parsed sort directive. Column is a validated
// logical name, never raw user input.
type orderByClause struct {
	Column string
	Desc   bool
}

// orderByColumnAllowList maps API sort fields onto the logical column names
// understood by the catalog query layer.
var orderByColumnAllowList = map[string]string{
	"name":         "name",
	"display_name": "name",
	"displayname":  "name",
	"version":      "version",
	"created_at":   "created_at",
	"createdat":    "created_at",
	"updated_at":   "created_at",
	"updatedat":    "created_at",
}

// parseOrderBy parses the order_by query string, e.g. "name, created_at DESC".
// Empty input yields the default ordering (created_at DESC).
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

		column, ok := orderByColumnAllowList[strings.ToLower(field)]
		if !ok {
			return nil, fmt.Errorf("unknown order_by field %q", field)
		}

		if _, dup := seen[column]; dup {
			return nil, fmt.Errorf("order_by field %q appears more than once", field)
		}

		seen[column] = struct{}{}
		clauses = append(clauses, orderByClause{Column: column, Desc: desc})
	}

	return clauses, nil
}

// defaultAgentOrder sorts newest-first when order_by is omitted.
func defaultAgentOrder() []orderByClause {
	return []orderByClause{{Column: "created_at", Desc: true}}
}

// encodePageToken serialises a positive offset into an opaque continuation
// token. A non-positive offset yields an empty token.
func encodePageToken(offset int) string {
	if offset <= 0 {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// decodePageToken parses a previously issued token. Empty input is valid and
// yields offset 0; malformed tokens are rejected.
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
