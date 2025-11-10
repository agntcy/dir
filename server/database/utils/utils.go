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

func QueryToFilters(queries []*searchv1.RecordQuery) ([]types.FilterOption, error) { //nolint:gocognit,cyclop
	var options []types.FilterOption

	for _, query := range queries {
		switch query.GetType() {
		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED:
			logger.Warn("Unspecified query type, skipping", "query", query)

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME:
			options = append(options, types.WithName(query.GetValue()))

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION:
			options = append(options, types.WithVersion(query.GetValue()))

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

		case searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE:
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

		default:
			logger.Warn("Unknown query type", "type", query.GetType())
		}
	}

	return options, nil
}
