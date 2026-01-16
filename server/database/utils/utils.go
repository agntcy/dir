// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"strconv"
	"strings"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("database/utils")

// ParseComparisonOperator parses a value that may have an operator prefix (>=, >, <=, <, =).
// Returns the operator and the actual value. If no operator prefix, returns empty operator.
func ParseComparisonOperator(value string) (string, string) {
	// Check for two-character operators first
	if op, found := strings.CutPrefix(value, ">="); found {
		return ">=", op
	}

	if op, found := strings.CutPrefix(value, "<="); found {
		return "<=", op
	}

	if op, found := strings.CutPrefix(value, ">"); found {
		return ">", op
	}

	if op, found := strings.CutPrefix(value, "<"); found {
		return "<", op
	}

	if op, found := strings.CutPrefix(value, "="); found {
		return "=", op
	}

	// No operator prefix
	return "", value
}

// BuildComparisonConditions builds SQL conditions for values with comparison operators.
// Only values with operator prefixes (>=, >, <=, <, =) are processed as comparisons (AND logic).
// Values without operators are processed as wildcards (OR logic).
// If both are present, they are combined with OR.
func BuildComparisonConditions(column string, values []string) (string, []any) {
	if len(values) == 0 {
		return "", nil
	}

	var comparisonConditions []string

	var comparisonArgs []any

	var wildcardValues []string

	// Separate comparison operators from regular values.
	for _, value := range values {
		op, actualValue := ParseComparisonOperator(value)
		if op != "" {
			comparisonConditions = append(comparisonConditions, fmt.Sprintf("%s %s ?", column, op))
			comparisonArgs = append(comparisonArgs, actualValue)
		} else {
			wildcardValues = append(wildcardValues, value)
		}
	}

	var allConditions []string

	var allArgs []any

	// Comparison conditions are AND'd together (e.g., >= 1.0 AND < 2.0).
	if len(comparisonConditions) > 0 {
		allConditions = append(allConditions, "("+strings.Join(comparisonConditions, " AND ")+")")
		allArgs = append(allArgs, comparisonArgs...)
	}

	// Wildcard conditions are OR'd together
	if len(wildcardValues) > 0 {
		wildcardCondition, wildcardArgs := BuildWildcardCondition(column, wildcardValues)
		if wildcardCondition != "" {
			allConditions = append(allConditions, "("+wildcardCondition+")")
			allArgs = append(allArgs, wildcardArgs...)
		}
	}

	if len(allConditions) == 0 {
		return "", nil
	}

	// If we have both comparison and wildcard, OR them together
	return strings.Join(allConditions, " OR "), allArgs
}

func QueryToFilters(queries []*searchv1.RecordQuery) ([]types.FilterOption, error) { //nolint:gocognit,cyclop
	var options []types.FilterOption

	for _, query := range queries {
		switch query.GetType() {
		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED:
			logger.Warn("Unspecified query type, skipping", "query", query)

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME:
			options = append(options, types.WithNames(query.GetValue()))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION:
			options = append(options, types.WithVersions(query.GetValue()))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID:
			u64, err := strconv.ParseUint(query.GetValue(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse skill ID %q: %w", query.GetValue(), err)
			}

			options = append(options, types.WithSkillIDs(u64))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME:
			options = append(options, types.WithSkillNames(query.GetValue()))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR:
			l := strings.SplitN(query.GetValue(), ":", 2) //nolint:mnd

			// If the type starts with a wildcard, treat it as a URL pattern
			// Example: "*marketing-strategy"
			if len(l) == 1 && strings.HasPrefix(l[0], "*") {
				options = append(options, types.WithLocatorURLs(l[0]))

				break
			}

			if len(l) == 1 && strings.TrimSpace(l[0]) != "" {
				options = append(options, types.WithLocatorTypes(l[0]))

				break
			}

			// If the prefix is //, check if the part before : is a wildcard
			// If it's a wildcard (like "*"), treat the whole thing as a URL pattern
			// If it's not a wildcard (like "docker-image"), treat as type:url format
			// Example: "*://ghcr.io/agntcy/marketing-strategy" -> pure URL pattern
			if len(l) == 2 && strings.HasPrefix(l[1], "//") && strings.HasPrefix(l[0], "*") {
				options = append(options, types.WithLocatorURLs(query.GetValue()))

				break
			}

			if len(l) == 2 { //nolint:mnd
				if strings.TrimSpace(l[0]) != "" {
					options = append(options, types.WithLocatorTypes(l[0]))
				}

				if strings.TrimSpace(l[1]) != "" {
					options = append(options, types.WithLocatorURLs(l[1]))
				}
			}

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME:
			if strings.TrimSpace(query.GetValue()) != "" {
				options = append(options, types.WithModuleNames(query.GetValue()))
			}

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_ID:
			u64, err := strconv.ParseUint(query.GetValue(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse domain ID %q: %w", query.GetValue(), err)
			}

			options = append(options, types.WithDomainIDs(u64))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME:
			options = append(options, types.WithDomainNames(query.GetValue()))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_CREATED_AT:
			options = append(options, types.WithCreatedAts(query.GetValue()))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR:
			options = append(options, types.WithAuthors(query.GetValue()))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION:
			options = append(options, types.WithSchemaVersions(query.GetValue()))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_ID:
			u64, err := strconv.ParseUint(query.GetValue(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse module ID %q: %w", query.GetValue(), err)
			}

			options = append(options, types.WithModuleIDs(u64))

		default:
			logger.Warn("Unknown query type", "type", query.GetType())
		}
	}

	return options, nil
}
